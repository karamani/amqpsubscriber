configure:
	gb vendor update --all

build:
	gofmt -w src/amqpsubscriber
	go tool vet src/amqpsubscriber/*.go
	golint src/amqpsubscriber
	gb test
	gb build
