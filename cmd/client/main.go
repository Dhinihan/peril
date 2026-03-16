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

	gameState := gamelogic.NewGameState(name)
	err = pubsub.SubscribeJSON(
		connection,
		routing.ExchangePerilDirect,
		queueName,
		routing.PauseKey,
		pubsub.TRANSIENT_QUEUE,
		handlerPause(gameState),
	)
	if err != nil {
		log.Fatalln(err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	fmt.Println("Server is running, press ctrl+c to stop")

	go func() {
		for {
			time.Sleep(10 * time.Millisecond)
			ip := gamelogic.GetInput()
			if len(ip) < 1 {
				continue
			}
			if ip[0] == "spawn" {
				fmt.Println("Spawning new unit")
				err := gameState.CommandSpawn(ip)
				if err != nil {
					fmt.Println(err)
				}
				continue
			}
			if ip[0] == "move" {
				fmt.Println("Moving units")
				_, err := gameState.CommandMove(ip)
				if err != nil {
					fmt.Println(err)
				}
				continue
			}
			if ip[0] == "status" {
				fmt.Println("Printing Status")
				gameState.CommandStatus()
				continue
			}
			if ip[0] == "help" {
				fmt.Println("Printing help")
				gamelogic.PrintClientHelp()
				continue
			}
			if ip[0] == "span" {
				fmt.Println("Spamming not allowed yet!")
				continue
			}
			if ip[0] == "quit" {
				gamelogic.PrintQuit()
				sigChan <- os.Interrupt
				continue
			}
			fmt.Println("unknown command")
		}
	}()

	<-sigChan
	fmt.Println("\n ctrl+c received, shutting down")

}

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) {
	return func(ps routing.PlayingState) {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
	}

}
