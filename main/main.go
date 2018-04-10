package main

import (
	"github.com/anabiozz/asgard/agent"
	"github.com/anabiozz/asgard/internal/config"
	_ "github.com/anabiozz/asgard/plugins/inputs/all"
	_ "github.com/anabiozz/asgard/plugins/outputs/all"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var stop chan struct{}

func loop(stop chan struct{}) {
	reload := make(chan bool, 1)
	reload <- true
	for <-reload {
		reload <- false

		// Create new config
		newConfig := config.NewConfig()
		// Filling new config getting data from default config
		err := newConfig.LoadConfig()
		if err != nil {
			log.Fatalf("ERROR: %s", err)
		}

		inputs := newConfig.InputFilters["inputs"].([]interface{})
		outputs := newConfig.OutputFilters["outputs"].([]interface{})

		// Filling InputFilters
		for _, value := range inputs {
			newConfig.AddInput(value.(string))
		}
		// Filling OutputFilters
		for _, value := range outputs {
			newConfig.AddOutput(value.(string))
		}

		if len(newConfig.Inputs) == 0 || len(newConfig.Outputs) == 0 {
			log.Fatalf("ERROR: no inputs or outputs found, did you provide a valid config file?")
		}

		// Create new agent with confing
		newAgent, err := agent.NewAgent(newConfig)
		if err != nil {
			log.Fatal("ERROR: " + err.Error())
		}

		err = newAgent.Connect()
		if err != nil {
			log.Fatal("ERROR: " + err.Error())
		}

		shutdown := make(chan struct{})
		signals := make(chan os.Signal)
		signal.Notify(signals, os.Interrupt, syscall.SIGHUP)
		go func() {
			select {
			case sig := <-signals:
				if sig == os.Interrupt {
					close(shutdown)
				}
				if sig == syscall.SIGHUP {
					log.Printf("I! Reloading config\n")
					<-reload
					reload <- true
					close(shutdown)
				}
			case <-stop:
				close(shutdown)
			}
		}()
		newAgent.Run(shutdown)
	}
}

func main() {
	// TODO: implement a feature that will be obtain all processes in system and to fill inputFilters
	stop = make(chan struct{})
	loop(stop)
}
