//
// https://gist.github.com/schmohlio/d7bdb255ba61d3f5e51a512a7c0d6a85
package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
        "bufio"
        "os/exec"
        "regexp"
        "io/ioutil"
        "strings"
        "crypto/x509"
        "crypto/tls"
        "os"
)

// the amount of time to wait when pushing a message to
// a slow client or a client that closed after `range clients` started.
const patience time.Duration = time.Second*1

// Example SSE server in Golang.
//     $ go run sse.go

//The BackLog
var backLogLength = 500
var backLog [][]byte = make([][]byte, 0, 2*backLogLength)
//Server-side filtering:
//var backLogFilenameMustContain = "_yournamespace_" //dont put system logs in the backlog

type Broker struct {

	// Events are pushed to this channel by the main events-gathering routine
	Notifier chan []byte

	// New client connections
	newClients chan chan []byte

	// Closed client connections
	closingClients chan chan []byte

	// Client connections registry
	clients map[chan []byte]bool

}

func NewServer() (broker *Broker) {
	// Instantiate a broker
	broker = &Broker{
		Notifier:       make(chan []byte, 1),
		newClients:     make(chan chan []byte),
		closingClients: make(chan chan []byte),
		clients:        make(map[chan []byte]bool),
	}

	// Set it running - listening and broadcasting events
	go broker.listen()

	return
}

func fetchAndWriteNamespaces(rw http.ResponseWriter) {
        rw.Header().Set("Content-Type", "application/json")
 
        ca := x509.NewCertPool()
        certs, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	if err != nil {
		log.Print("Error: %v", err)
                return;
	}
	// Append our cert to the pool
	if ok := ca.AppendCertsFromPEM(certs); !ok {
		log.Print("Error: %v", ok)
                return;
	}

	// Trust the cert pool in our client
	config := &tls.Config{
		RootCAs:            ca,
	}
	tr := &http.Transport{TLSClientConfig: config}
	client := &http.Client{Transport: tr}

        url := "https://"+os.Getenv("KUBERNETES_PORT_443_TCP_ADDR")+":"+os.Getenv("KUBERNETES_PORT_443_TCP_PORT")+"/api/v1/namespaces/"
	req, _ := http.NewRequest("GET", url, nil)

        //Read token file
        token, _ := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")

        req.Header.Set("Authorization", "Bearer "+string(token))
	resp, err := client.Do(req)

        if err != nil {
		log.Print("Error: %v", err)
                return
        }

        defer resp.Body.Close()
        body, err := ioutil.ReadAll(resp.Body)
        if err != nil {
		log.Print("Error: %v", err)
                return
        }

        rw.Write(body);

        return      
}

func (broker *Broker) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

        if (req.URL.Path == "/")  {
            rw.Header().Set("Content-Type", "text/html")
            dat, _ := ioutil.ReadFile("/index.html")
            //TODO: error handling
            rw.Write(dat)
            return
        }
        if (req.URL.Path == "/namespaces")  {
            fetchAndWriteNamespaces(rw)
            //TODO: error handling
            return
        }
        if (req.URL.Path == "/debug")  {
            rw.Header().Set("Content-Type", "text/plain")
            fmt.Fprintf(rw, "length = %d\n", len(backLog))

            return
        }

	// Make sure that the writer supports flushing.
	//
	flusher, ok := rw.(http.Flusher)

	if !ok {
		http.Error(rw, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")
	rw.Header().Set("Access-Control-Allow-Origin", "*")

	// Each connection registers its own message channel with the Broker's connections registry
	messageChan := make(chan []byte)

	// Signal the broker that we have a new connection
	broker.newClients <- messageChan

	// Remove this client from the map of connected clients
	// when this handler exits.
	defer func() {
		broker.closingClients <- messageChan
	}()

	// Listen to connection close and un-register messageChan
	notify := rw.(http.CloseNotifier).CloseNotify()

        //dump the backLog
        for _, element := range backLog  {
	    fmt.Fprintf(rw, "data: %s\n\n", element)
        }
	flusher.Flush()

	for {
		select {
		case <-notify:
			return
		default:

			// Write to the ResponseWriter
			// Server Sent Events compatible
			fmt.Fprintf(rw, "data: %s\n\n", <-messageChan)

			// Flush the data immediatly instead of buffering it for later.
			flusher.Flush()
		}
	}

}

func (broker *Broker) listen() {
	for {
		select {
		case s := <-broker.newClients:

			// A new client has connected.
			// Register their message channel
			broker.clients[s] = true
			log.Printf("Client added. %d registered clients", len(broker.clients))
		case s := <-broker.closingClients:

			// A client has dettached and we want to
			// stop sending them messages.
			delete(broker.clients, s)
			log.Printf("Removed client. %d registered clients", len(broker.clients))
		case event := <-broker.Notifier:

			// We got a new event from the outside!
			// Send event to all connected clients
			for clientMessageChan, _ := range broker.clients {
				select {
				case clientMessageChan <- event:
				case <-time.After(patience):
					log.Print("Skipping client.")
				}
			}
		}
	}

}

func main() {

	broker := NewServer()

        cmd := exec.Command("/xtail","/var/log/containers")

        stdout, err := cmd.StdoutPipe()
        checkError(err)
        err = cmd.Start()
        checkError(err)
        defer cmd.Wait()  // Doesn't block
 
        scanner := bufio.NewScanner(stdout)
 
        currentFile := ""
        re1 := regexp.MustCompile("^\\*\\*\\* /var/log/containers/(?P<Path>.*) \\*\\*\\*$")

	go func() {
              for scanner.Scan() {
                      eventString := scanner.Text()
                      m1 := re1.FindStringSubmatch(eventString)
                      if (m1 != nil)  {
                          currentFile = m1[1]
                          //log.Println("File: "+currentFile)
                      } else if (eventString != "") && (! strings.HasPrefix(eventString, "***"))  {
                          jsonBytes := []byte("{\"fileName\":\""+currentFile+"\",\"logObject\":"+eventString+"}")

                          //log.Println("Receiving event")
                          broker.Notifier <- jsonBytes
                          //put it on the backlog
                          //if (strings.Contains(currentFile, backLogFilenameMustContain))  {
                          backLog = append(backLog, jsonBytes)
                          if (len(backLog) >= backLogLength)   {
                              //shift
                              backLog = backLog[1:]
                          }
                          //}
                      } else {
                          //fmt.Println("Ignoring line")
                      }
              }


	}()

	log.Fatal("HTTP server error: ", http.ListenAndServe(":3000", broker))

}

func checkError(err error) {
    if err != nil {
        log.Fatalf("Error: %s", err)
    }
}

