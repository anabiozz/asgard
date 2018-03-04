package main

import (
	"fmt"
	"heimdall_project/asgard/agent"
	"heimdall_project/asgard/internal/config"
	"heimdall_project/asgard/plugins/inputs"
	_ "heimdall_project/asgard/plugins/inputs/all"
	"heimdall_project/asgard/plugins/outputs"
	_ "heimdall_project/asgard/plugins/outputs/all"
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

		// Create new config
		conf := config.NewConfig()

		// TODO: implement config filling
		conf.InputFilters = inputFilters
		conf.OutputFilters = outputFilters

		for _, value := range conf.InputFilters {
			conf.AddInput(value)
		}
		for _, value := range conf.OutputFilters {
			conf.AddOutput(value)
		}

		if len(conf.Inputs) == 0 {
			log.Fatalf("E! Error: no inputs found, did you provide a valid config file?")
		}
		if len(conf.Outputs) == 0 {
			log.Fatalf("E! Error: no outputs found, did you provide a valid config file?")
		}

		// Create new agent with confing
		ag, err := agent.NewAgent(conf)
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

	// TODO: implement a feature that will be obtain all processes in system
	// and to fill inputFilters

	inputFilters, outputFilters := []string{}, []string{}
	inputFilters = append(inputFilters, "cpu")
	inputFilters = append(inputFilters, "mem")
	outputFilters = append(outputFilters, "influxdb")

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
