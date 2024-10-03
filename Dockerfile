# --- Build stage ---
#
FROM golang:1.22-alpine AS builder

COPY serve.go /go
RUN go mod init example.com/m/v2
RUN CGO_ENABLED=0 go build -a -installsuffix cgo -o serve

#--- Final stage ---

FROM debian:stable

COPY --from=builder /go/serve /serve
RUN apt-get update && apt-get install -y xtail
COPY index.html /

ENTRYPOINT /serve

