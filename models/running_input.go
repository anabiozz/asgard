package models

import (
	"heimdall_project/asgard"
	"time"
	"fmt"
)

type RunningInput struct {
	Input  asgard.Input
	Config *InputConfig
	trace       bool
	defaultTags map[string]string
}

// InputConfig containing a name, interval, and filter
type InputConfig struct {
	Name              string
	NameOverride      string
	MeasurementPrefix string
	MeasurementSuffix string
	Tags              map[string]string
	Interval          time.Duration
}

func (r *RunningInput) Name() string {
	return "inputs." + r.Config.Name
}

// MakeMetric either returns a metric, or returns nil if the metric doesn't
// need to be created (because of filtering, an error, etc.)
func (r *RunningInput) MakeMetric(
	measurement string,
	fields map[string]interface{},
	tags map[string]string,
	mType asgard.ValueType,
	t time.Time,
) asgard.Metric {
	m := makemetric(
		measurement,
		fields,
		tags,
		mType,
		t,
	)

	if r.trace && m != nil {
		fmt.Print("> " + m.String())
	}

	return m
}

func NewRunningInput(input asgard.Input) *RunningInput {
	return &RunningInput{
		Input:  input,
	}
}
