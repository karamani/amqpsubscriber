package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/streadway/amqp"
	"log"
	"os"
	"strconv"
)

var (
	debugMode            bool
	amqpConnectionString string
	exchangeName         string
	exchangeDurable      bool
	exchangeAutodelete   bool
	exchangeExclusive    bool
	exchangeNoWait       bool
	queueName            string
	unquote              bool
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s\n", msg, err)
	}
}

func main() {

	app := cli.NewApp()
	app.Name = "AMQP Subscriber"
	app.Usage = "AMQP Subscriber."
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:        "debug",
			Usage:       "debug mode",
			EnvVar:      "AMQP_SUBSCRIBERD_DEBUG",
			Destination: &debugMode,
		},
	}
	app.Action = func(c *cli.Context) {
		log.Println("see usage!")
	}
	app.Commands = []cli.Command{
		{
			Name:  "subscribe",
			Usage: "подписаться на результаты генерации прайсов",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "amqp",
					Usage:       "amqp connection string",
					EnvVar:      "AMQPSUBSCRIBER_CNN",
					Destination: &amqpConnectionString,
				},
				cli.StringFlag{
					Name:        "exchange",
					Usage:       "exchange name",
					EnvVar:      "AMQPSUBSCRIBER_EXCHANGE",
					Destination: &exchangeName,
				},
				cli.BoolFlag{
					Name:        "exchange_durable",
					Usage:       "exchange durable flag",
					EnvVar:      "AMQPSUBSCRIBER_EXCHANGE_DURABLE",
					Destination: &exchangeDurable,
				},
				cli.BoolFlag{
					Name:        "exchange_autodelete",
					Usage:       "exchange autodelete flag",
					EnvVar:      "AMQPSUBSCRIBER_EXCHANGE_AUTODELETE",
					Destination: &exchangeAutodelete,
				},
				cli.BoolFlag{
					Name:        "exchange_exclusive",
					Usage:       "exchange exclusive",
					EnvVar:      "AMQPSUBSCRIBER_EXCHANGE_EXCLUSIVE",
					Destination: &exchangeExclusive,
				},
				cli.BoolFlag{
					Name:        "exchange_nowait",
					Usage:       "exchange no-wait flag",
					EnvVar:      "AMQPSUBSCRIBER_EXCHANGE_NOWAIT",
					Destination: &exchangeNoWait,
				},
				cli.StringFlag{
					Name:        "queue",
					Usage:       "queue name",
					EnvVar:      "AMQPSUBSCRIBER_QUEUE",
					Destination: &queueName,
				},
				cli.BoolFlag{
					Name:        "unquote",
					Usage:       "unquote message body",
					Destination: &unquote,
				},
			},
			Action: func(c *cli.Context) {
				handleResults()
			},
		},
	}

	app.Run(os.Args)
}

func handleResults() {

	conn, err := amqp.Dial(amqpConnectionString)
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	err = ch.ExchangeDeclare(
		exchangeName,
		"fanout",
		exchangeDurable,
		exchangeAutodelete,
		exchangeExclusive,
		exchangeNoWait,
		nil,
	)
	failOnError(err, "Failed to declare an exchange")

	err = ch.Qos(1, 0, false)
	failOnError(err, "Qos error")

	durable := len(queueName) > 0
	deleteWhenUnused := len(queueName) == 0

	q, err := ch.QueueDeclare(
		queueName,
		durable,
		deleteWhenUnused,
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	failOnError(err, "Failed to declare a queue")

	err = ch.QueueBind(
		q.Name,
		"",
		exchangeName,
		false,
		nil,
	)
	failOnError(err, "Failed to bind a queue")

	msgs, err := ch.Consume(
		q.Name,
		"",    // consumer
		true,  // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	failOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		var body string
		for d := range msgs {
			body = string(d.Body)
			if unquote {
				body = "\"" + body + "\""
				body, err = strconv.Unquote(body)
				if err != nil {
					log.Println(err.Error())
				}
			}
			fmt.Printf("%s\n", body)
		}
		forever <- true
	}()

	select {
	case amqpErr := <-ch.NotifyClose(make(chan *amqp.Error)):
		log.Fatalln("NotifyClose: " + amqpErr.Error())
	case <-forever:
		log.Println("forever = true")
	}
}
