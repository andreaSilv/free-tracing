docker-build:
	docker build -t <image>/app:latest .

docker-push:
	docker push <image>/app:latest

docker-bp: docker-build docker-push

run:
	go run .

build:
	go build .
