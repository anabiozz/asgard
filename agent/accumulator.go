package agent

import (
	"github.com/anabiozz/asgard"
	"time"
)

type accumulator struct {
	metrics   chan asgard.Metric
	maker     MetricMaker
	precision time.Duration
}

// MetricMaker ...
type MetricMaker interface {
	MakeMetric(
		measurement string,
		fields map[string]interface{},
		tags map[string]string,
		mType asgard.ValueType,
		t time.Time) asgard.Metric
}

// NewAccumulator ...
func NewAccumulator(
	runningInput MetricMaker,
	metrics chan asgard.Metric) *accumulator {

	acc := accumulator{
		maker:     runningInput,
		metrics:   metrics,
		precision: time.Nanosecond,
	}
	return &acc
}

func (ac *accumulator) AddFields(
	measurement string,
	fields map[string]interface{},
	tags map[string]string,
	t ...time.Time) {

	if m := ac.maker.MakeMetric(measurement, fields, tags, asgard.Untyped, ac.getTime(t)); m != nil {
		ac.metrics <- m
	}
}

func (ac *accumulator) AddGauge(
	measurement string,
	fields map[string]interface{},
	tags map[string]string,
	t ...time.Time) {

	if m := ac.maker.MakeMetric(measurement, fields, tags, asgard.Gauge, ac.getTime(t)); m != nil {
		ac.metrics <- m
	}
}

func (ac *accumulator) AddCounter(
	measurement string,
	fields map[string]interface{},
	tags map[string]string,
	t ...time.Time) {

	if m := ac.maker.MakeMetric(measurement, fields, tags, asgard.Counter, ac.getTime(t)); m != nil {
		ac.metrics <- m
	}
}

func (ac *accumulator) AddSummary(
	measurement string,
	fields map[string]interface{},
	tags map[string]string,
	t ...time.Time) {

	if m := ac.maker.MakeMetric(measurement, fields, tags, asgard.Summary, ac.getTime(t)); m != nil {
		ac.metrics <- m
	}
}

func (ac *accumulator) AddHistogram(
	measurement string,
	fields map[string]interface{},
	tags map[string]string,
	t ...time.Time) {

	if m := ac.maker.MakeMetric(measurement, fields, tags, asgard.Histogram, ac.getTime(t)); m != nil {
		ac.metrics <- m
	}
}

// AddError passes a runtime error to the accumulator.
// The error will be tagged with the plugin name and written to the log.
func (ac *accumulator) AddError(err error) {
	if err == nil {
		return
	}
}

func (ac accumulator) getTime(t []time.Time) time.Time {
	var timestamp time.Time
	if len(t) > 0 {
		timestamp = t[0]
	} else {
		timestamp = time.Now()
	}
	return timestamp.Round(ac.precision)
}
