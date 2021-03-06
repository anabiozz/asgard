package serializers

import (
	"fmt"
	"github.com/anabiozz/asgard"
	"github.com/anabiozz/asgard/plugins/serializers/influx"
	"github.com/anabiozz/asgard/plugins/serializers/json"
	"time"
)

// SerializerOutput is an interface for output plugins that are able to
// serialize  metrics into arbitrary data formats.
type SerializerOutput interface {
	// SetSerializer sets the serializer function for the interface.
	SetSerializer(serializer Serializer)
}

// Serializer is an interface defining functions that a serializer plugin must
// satisfy.
type Serializer interface {
	// Serialize takes a single  metric and turns it into a byte buffer.
	// separate metrics should be separated by a newline, and there should be
	// a newline at the end of the buffer.
	Serialize(metric asgard.Metric) ([]byte, error)
}

// Config is a struct that covers the data types needed for all serializer types,
// and can be used to instantiate _any_ of the serializers.
type Config struct {
	// Dataformat can be one of: influx, graphite, or json
	DataFormat string

	// Prefix to add to all measurements, only supports Graphite
	Prefix string

	// Template for converting metrics into Graphite
	// only supports Graphite
	Template string

	// Timestamp units to use for JSON formatted output
	TimestampUnits time.Duration
}

// NewSerializer a Serializer interface based on the given config.
func NewSerializer(config *Config) (Serializer, error) {
	var err error
	var serializer Serializer
	switch config.DataFormat {
	case "influx":
		serializer, err = NewInfluxSerializer()
	case "json":
		serializer, err = NewJsonSerializer(config.TimestampUnits)
	default:
		err = fmt.Errorf("Invalid data format: %s", config.DataFormat)
	}
	return serializer, err
}

func NewJsonSerializer(timestampUnits time.Duration) (Serializer, error) {
	return &json.JsonSerializer{TimestampUnits: timestampUnits}, nil
}

func NewInfluxSerializer() (Serializer, error) {
	return &influx.InfluxSerializer{}, nil
}
