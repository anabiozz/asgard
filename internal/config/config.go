package config

import (
	"fmt"
	"heimdall_project/asgard/internal/models"
	"heimdall_project/asgard/plugins/inputs"
	"heimdall_project/asgard/plugins/outputs"
	"heimdall_project/asgard/plugins/serializers"
	"time"
)

type AgentConfig struct {
	// Interval at which to gather information
	Interval time.Duration

	// RoundInterval rounds collection interval to 'interval'.
	//     ie, if Interval=10s then always collect on :00, :10, :20, etc.
	RoundInterval bool

	// By default or when set to "0s", precision will be set to the same
	// timestamp order as the collection interval, with the maximum being 1s.
	//   ie, when interval = "10s", precision will be "1s"
	//       when interval = "250ms", precision will be "1ms"
	// Precision will NOT be used for service inputs. It is up to each individual
	// service input to set the timestamp at the appropriate precision.
	Precision int

	// CollectionJitter is used to jitter the collection by a random amount.
	// Each plugin will sleep for a random time within jitter before collecting.
	// This can be used to avoid many plugins querying things like sysfs at the
	// same time, which can have a measurable effect on the system.
	CollectionJitter int

	// FlushInterval is the Interval at which to flush data
	FlushInterval time.Duration

	// FlushJitter Jitters the flush interval by a random amount.
	// This is primarily to avoid large write spikes for users running a large
	// number of  instances.
	// ie, a jitter of 5s and interval 10s means flushes will happen every 10-15s
	FlushJitter int

	// MetricBatchSize is the maximum number of metrics that is wrote to an
	// output plugin in one call.
	MetricBatchSize int

	// MetricBufferLimit is the max number of metrics that each output plugin
	// will cache. The buffer is cleared when a successful write occurs. When
	// full, the oldest metrics will be overwritten. This number should be a
	// multiple of MetricBatchSize. Due to current implementation, this could
	// not be less than 2 times MetricBatchSize.
	MetricBufferLimit int

	// FlushBufferWhenFull tells  to flush the metric buffer whenever
	// it fills up, regardless of FlushInterval. Setting this option to true
	// does _not_ deactivate FlushInterval.
	FlushBufferWhenFull bool

	// Debug is the option for running in debug mode
	Debug bool

	// Logfile specifies the file to send logs to
	Logfile string

	// Quiet is the option for running in quiet mode
	Quiet        bool
	Hostname     string
	OmitHostname bool
}

// Config ...
type Config struct {
	Tags          map[string]string
	InputFilters  []string
	OutputFilters []string

	Agent   *AgentConfig
	Inputs  []*models.RunningInput
	Outputs []*models.RunningOutput
}

// NewConfig ...
func NewConfig() *Config {
	c := &Config{
		Agent: &AgentConfig{
			Interval:      time.Duration(time.Millisecond * 10000),
			RoundInterval: false,
			FlushInterval: time.Duration(time.Millisecond * 10000),
			// OmitHostname: false,
		},
		Tags:    make(map[string]string),
		Inputs:  make([]*models.RunningInput, 0),
		Outputs: make([]*models.RunningOutput, 0),
	}
	return c
}

// Check the occurrence of the name in list array
func sliceContains(name string, list []string) bool {
	for _, b := range list {
		if b == name {
			return true
		}
	}
	return false
}

// AddInput ...
func (c *Config) AddInput(name string) error {
	if len(c.InputFilters) > 0 && !sliceContains(name, c.InputFilters) {
		return nil
	}
	// Legacy support renaming io input to diskio
	if name == "io" {
		name = "diskio"
	}

	creator, ok := inputs.Inputs[name]
	if !ok {
		return fmt.Errorf("Undefined but requested input: %s", name)
	}
	input := creator()

	rp := models.NewRunningInput(input)
	c.Inputs = append(c.Inputs, rp)
	return nil
}

// AddOutput ...
func (c *Config) AddOutput(name string) error {
	if len(c.OutputFilters) > 0 && !sliceContains(name, c.OutputFilters) {
		return nil
	}
	creator, ok := outputs.Outputs[name]
	if !ok {
		return fmt.Errorf("Undefined but requested output: %s", name)
	}
	output := creator()

	switch t := output.(type) {
	case serializers.SerializerOutput:
		serializer, err := buildSerializer("json")
		if err != nil {
			return err
		}
		t.SetSerializer(serializer)
	}

	ro := models.NewRunningOutput(name, output, c.Agent.MetricBatchSize, c.Agent.MetricBufferLimit)
	c.Outputs = append(c.Outputs, ro)
	return nil
}

// buildSerializer grabs the necessary entries from the ast.Table for creating
// a serializers.Serializer object, and creates it, which can then be added onto
// an Output object.
func buildSerializer(dataFormat string) (serializers.Serializer, error) {
	c := &serializers.Config{TimestampUnits: time.Duration(10 * time.Second)}

	c.DataFormat = dataFormat
	if c.DataFormat == "" {
		c.DataFormat = "json"
	}

	return serializers.NewSerializer(c)
}
