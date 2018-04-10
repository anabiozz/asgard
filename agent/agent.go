package agent

import (
	"fmt"
	"github.com/anabiozz/asgard"
	"github.com/anabiozz/asgard/internal/config"
	"github.com/anabiozz/asgard/internal/models"
	"log"
	"os"
	"sync"
	"time"
)

// Agent ...
type Agent struct {
	Config *config.Config
}

// NewAgent returns an Agent struct based off the given Config
func NewAgent(config *config.Config) (*Agent, error) {

	a := &Agent{
		Config: config,
	}
	if !a.Config.Agent.OmitHostname {
		if a.Config.Agent.Hostname == "" {
			hostname, err := os.Hostname()
			if err != nil {
				return nil, err
			}
			a.Config.Agent.Hostname = hostname
		}

		config.Tags["host"] = a.Config.Agent.Hostname
	}

	return a, nil
}

// gatherWithTimeout gathers from the given input, with the given timeout.
//   when the given timeout is reached, gatherWithTimeout logs an error message
//   but continues waiting for it to return. This is to avoid leaving behind
//   hung processes, and to prevent re-calling the same hung process over and
//   over.
func gatherWithTimeout(
	shutdown chan struct{},
	input *models.RunningInput,
	acc *accumulator,
	timeout time.Duration) {

	ticker := time.NewTicker(timeout)
	defer ticker.Stop()
	done := make(chan error)

	go func() {
		done <- input.Input.Gather(acc)
	}()

	for {
		select {
		case err := <-done:
			if err != nil {
				acc.AddError(err)
			}
			return
		case <-ticker.C:
			err := fmt.Errorf("took longer to collect than collection interval (%s)", timeout)
			acc.AddError(err)
			continue
		case <-shutdown:
			return
		}
	}
}

// flusher monitors the metrics input channel and flushes on the minimum interval
func (a *Agent) flusher(
	shutdown chan struct{},
	metricC chan asgard.Metric) error {

	time.Sleep(time.Millisecond * 300)

	outMetricC := make(chan asgard.Metric, 100)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-shutdown:
				if len(outMetricC) > 0 {
					// hold on while outMetricC full
					continue
				}
				return
			case m := <-outMetricC:
				for i, o := range a.Config.Outputs {
					if i == len(a.Config.Outputs)-1 {
						o.AddMetric(m)
					} else {
						o.AddMetric(m.Copy())
					}
				}
			}
		}
	}()

	ticker := time.NewTicker(time.Duration(a.Config.Agent.FlushInterval * time.Millisecond))
	semaphore := make(chan struct{}, 1)

	for {
		select {
		case <-shutdown:
			log.Println("INFO: Hang on, flushing any cached metrics before shutdown")
			// wait for outMetricC to get flushed before flushing outputs
			wg.Wait()
			a.flush()
			return nil
		case <-ticker.C:
			go func() {
				select {
				case semaphore <- struct{}{}:
					a.flush()
					<-semaphore
				default:
					// skipping this flush because one is already happening
					log.Println("INFO: Skipping a scheduled flush because there is already a flush ongoing.")
				}
			}()
		case metric := <-metricC:
			// NOTE potential bottleneck here as we put each metric through the processors serially.
			mS := []asgard.Metric{metric}
			for _, m := range mS {
				outMetricC <- m
			}
		}
	}
}

// flush writes a list of metrics to all configured outputs
func (a *Agent) flush() {

	var wg sync.WaitGroup
	wg.Add(len(a.Config.Outputs))
	for _, o := range a.Config.Outputs {
		go func(output *models.RunningOutput) {
			defer wg.Done()
			err := output.Write()
			if err != nil {
				log.Printf("ERROR: Error writing to output [%s]: %s\n", output.Name, err.Error())
			}
		}(o)
	}
	wg.Wait()
}

// gatherer runs the inputs that have been configured with their own reporting interval.
func (a *Agent) gatherer(
	shutdown chan struct{},
	input *models.RunningInput,
	interval time.Duration,
	metricC chan asgard.Metric) {

	// Create new accumulator
	acc := NewAccumulator(input, metricC)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		gatherWithTimeout(shutdown, input, acc, interval)
		select {
		case <-shutdown:
			return
		case <-ticker.C:
			continue
		}
	}
}

// Close closes the connection to all configured outputs
func (a *Agent) Close() error {
	var err error
	for _, o := range a.Config.Outputs {
		err = o.Output.Close()
	}
	return err
}

// Connect connects to all configured outputs
func (a *Agent) Connect() error {
	for _, o := range a.Config.Outputs {
		log.Printf("DEBUG: Attempting connection to output: %s\n", o.Name)
		err := o.Output.Connect()
		if err != nil {
			log.Printf("ERROR: Failed to connect to output %s, retrying in 15s, error was '%s' \n", o.Name, err)
			time.Sleep(15 * time.Second)
			err = o.Output.Connect()
			if err != nil {
				return err
			}
		}
		log.Printf("DEBUG: Successfully connected to output: %s\n", o.Name)
	}
	return nil
}

// Run runs the agent daemon, gathering every Interval
func (a *Agent) Run(shutdown chan struct{}) error {
	var wg sync.WaitGroup

	// channel shared between all input threads for accumulating metrics
	metricChannel := make(chan asgard.Metric, 100)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := a.flusher(shutdown, metricChannel); err != nil {
			log.Printf("ERROR Flusher routine failed, exiting: %s\n", err.Error())
			close(shutdown)
		}
	}()

	wg.Add(len(a.Config.Inputs))
	for _, input := range a.Config.Inputs {

		// Set gatherer interval
		interval := time.Duration(a.Config.Agent.Interval * time.Millisecond)

		go func(input *models.RunningInput, interval time.Duration) {
			defer wg.Done()
			a.gatherer(shutdown, input, interval, metricChannel)
		}(input, interval)
	}

	wg.Wait()
	a.Close()
	return nil
}
