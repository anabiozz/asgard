package main

import (
	"log"
	"os"
	"syscall"
	"heimdall_project/asgard/internal/config"
	_ "heimdall_project/asgard/plugins/inputs/system"
	_ "heimdall_project/asgard/plugins/outputs/kafka"
	"heimdall_project/asgard/agent"
	"os/signal"
	"fmt"
	"heimdall_project/asgard/plugins/outputs"
	"heimdall_project/asgard/plugins/inputs"
)
var stop chan struct{}

func reloadLoop(stop chan struct{}, inputFilters []string, outputFilters []string) {
	reload := make(chan bool, 1)
	reload <- true
	for <-reload {
		reload <- false
		// If no other options are specified, load the config file and run.
		c := config.NewConfig()
		c.InputFilters = inputFilters
		c.OutputFilters = outputFilters

		for _, value := range c.InputFilters  {
			c.AddInput(value)
		}
		for _, value := range c.OutputFilters {
			c.AddOutput(value)
		}

		if len(c.Inputs) == 0 {
			log.Fatalf("E! Error: no inputs found, did you provide a valid config file?")
		}

		if len(c.Outputs) == 0 {
			log.Fatalf("E! Error: no outputs found, did you provide a valid config file?")
		}

		ag, err := agent.NewAgent(c)
		if err != nil {
			log.Fatal("E! " + err.Error())
		}

		err = ag.Connect()
		if err != nil {
			log.Fatal("E! " + err.Error())
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
					log.Printf("I! Reloading Telegraf config\n")
					<-reload
					reload <- true
					close(shutdown)
				}
			case <-stop:
				close(shutdown)
			}
		}()

		ag.Run(shutdown)
	}
}

func main() {
	inputFilters, outputFilters := []string{}, []string{}
	inputFilters = append(inputFilters, "cpu")
	outputFilters = append(outputFilters, "kafka")

	fmt.Println("Available Output Plugins:")
	for k, _ := range outputs.Outputs {
		fmt.Printf("  %s\n", k)
	}

	fmt.Println("Available Input Plugins:")
	for k, _ := range inputs.Inputs {
		fmt.Printf("  %s\n", k)
	}
	stop = make(chan struct{})
	reloadLoop(stop, inputFilters, outputFilters)
}