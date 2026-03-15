package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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
