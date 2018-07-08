# minikube-log-viewer
Lightweight Minikube Log Viewer

![minikube-log-viewer-screenshot.png](minikube-log-viewer-screenshot.png)

Usage (Mac + Linux):
```
kubectl create -f https://raw.githubusercontent.com/ivans3/minikube-log-viewer/master/deployment-and-service.yaml
echo The URL is http://$(minikube ip):32000/
```
And then visit the URL with your browser.

Features:
 * uses HTTP SSE
 * uses xtail as the log collector

TODO:
 * search feature
 * pause/resume feature
 * namespace filtering

