package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Dhinihan/peril/internal/gamelogic"
	"github.com/Dhinihan/peril/internal/pubsub"
	"github.com/Dhinihan/peril/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril server...")
	connectionUrl := "amqp://guest:guest@localhost:5672/"
	connection, err := amqp.Dial(connectionUrl)
	if err != nil {
		log.Fatalln(err)
	}
	defer connection.Close()
	fmt.Println("Connected to message broker")
	ch, err := connection.Channel()
	if err != nil {
		log.Fatalln(err)
	}
	defer ch.Close()

	pubsub.DeclareAndBind(
		connection,
		routing.ExchangePerilTopic,
		routing.GameLogSlug,
		routing.GameLogSlug+".*",
		pubsub.SQTDurable,
	)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	fmt.Println("Server is running, press ctrl+c to stop")
	gamelogic.PrintServerHelp()

	go func() {
		for {
			time.Sleep(10 * time.Millisecond)
			ip := gamelogic.GetInput()
			if len(ip) < 1 {
				continue
			}
			if ip[0] == "pause" {
				fmt.Println("sending pause msg")
				st := routing.PlayingState{IsPaused: true}
				pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, st)
				continue
			}
			if ip[0] == "resume" {
				fmt.Println("sending resume msg")
				st := routing.PlayingState{IsPaused: false}
				pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, st)
				continue
			}
			if ip[0] == "quit" {
				fmt.Println("exiting")
				sigChan <- os.Interrupt
				continue
			}
			fmt.Println("unknown command")
		}
	}()

	<-sigChan
	fmt.Println("\nshutting down")
}
