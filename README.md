# minikube-log-viewer
Lightweight Minikube Log Viewer

![minikube-log-viewer-screenshot.png](minikube-log-viewer-screenshot.png)

Installation

Available as a Minikube Add-On

```
minikube addons enable logviewer
```


```
echo The URL is http://$(minikube ip):32000/
```

And then visit the URL with your browser.

Features:
 * uses HTTP SSE (no indexer or indexing delay)
 * uses xtail as the log collector
 * namespace filtering (and you can bookmark a link with a `?namespace=yournamespace` query string to save it)
 * search feature
 * pause/resume feature
 * supports docker(JSON) and containerd log formats

TODO:
 * hilight matches in search feature
 * "Mark" Button which adds HBar to log stream...

