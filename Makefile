docker-build:
	docker build -t 10.40.3.175:5050/andrea.silvi/golang-opentelemetry/kubernetes_sidecar/sidecar:latest .

docker-push:
	docker push 10.40.3.175:5050/andrea.silvi/golang-opentelemetry/kubernetes_sidecar/sidecar:latest

docker-bp: docker-build docker-push

run:
	go run .

build:
	go build .
