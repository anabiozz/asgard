package main

import (
	"fmt"
	"heimdall_project/asgard/agent"
	"heimdall_project/asgard/internal/config"
	"heimdall_project/asgard/plugins/inputs"
	_ "heimdall_project/asgard/plugins/inputs/system"
	"heimdall_project/asgard/plugins/outputs"
	_ "heimdall_project/asgard/plugins/outputs/kafka"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var stop chan struct{}

func reloadLoop(stop chan struct{}, inputFilters []string, outputFilters []string) {
	reload := make(chan bool, 1)
	reload <- true
	for <-reload {
		reload <- false
		c := config.NewConfig()
		c.InputFilters = inputFilters
		c.OutputFilters = outputFilters

		for _, value := range c.InputFilters {
			c.AddInput(value)
		}
		for _, value := range c.OutputFilters {
			c.AddOutput(value)
		}

		if len(c.Inputs) == 0 {
			log.Fatalf("E! Error: no inputs found, did you provide a valid config file?")
		} else if len(c.Outputs) == 0 {
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
					log.Printf("I! Reloading config\n")
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
	for k := range outputs.Outputs {
		fmt.Printf("  %s\n", k)
	}

	fmt.Println("Available Input Plugins:")
	for k := range inputs.Inputs {
		fmt.Printf("  %s\n", k)
	}
	stop = make(chan struct{})
	reloadLoop(stop, inputFilters, outputFilters)
}
