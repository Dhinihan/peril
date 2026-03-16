package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Dhinihan/peril/internal/gamelogic"
	"github.com/Dhinihan/peril/internal/pubsub"
	"github.com/Dhinihan/peril/internal/routing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril client...")
	name, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalln("Error getting username")
	}
	connectionUrl := "amqp://guest:guest@localhost:5672/"
	connection, err := amqp.Dial(connectionUrl)
	if err != nil {
		log.Fatalln(err)
	}
	defer connection.Close()
	fmt.Println("Connected to message broker")
	queueName := strings.Join([]string{routing.PauseKey, name}, ".")
	_, _, err = pubsub.DeclareAndBind(
		connection,
		routing.ExchangePerilDirect,
		queueName,
		routing.PauseKey,
		pubsub.TRANSIENT_QUEUE,
	)
	if err != nil {
		log.Fatalln(err)
	}
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	fmt.Println("Server is running, press ctrl+c to stop")

	go func() {
		for {
			time.Sleep(1 * time.Second)
		}
	}()

	<-sigChan
	fmt.Println("\n ctrl+c received, shutting down")

}
