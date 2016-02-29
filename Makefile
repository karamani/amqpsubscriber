configure:
	gb vendor update --all

build:
	gofmt -w src/amqpsubscriber
	go tool vet src/amqpsubscriber/*.go
	gb test
	gb build