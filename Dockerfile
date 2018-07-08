FROM alpine:latest

COPY minikube-log-viewer /
COPY xtail /
COPY index.html /

ENTRYPOINT /minikube-log-viewer

