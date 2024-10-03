docker buildx create --name multiarch --bootstrap
#TODO: add other platforms:
docker buildx build --push  --builder multiarch --platform linux/arm64/v8,linux/amd64 -t ivans3/minikube-log-viewer:v1 .
docker buildx rm multiarch


