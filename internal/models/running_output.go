package models

import (
	"heimdall_project/asgard"
	"heimdall_project/asgard/internal/buffer"
	"log"
	"sync"
	"time"
	//"heimdall_project/asgard/metric"
	"fmt"
)

const (
	// Default size of metrics batch size.
	DEFAULT_METRIC_BATCH_SIZE = 1000

	// Default number of metrics kept. It should be a multiple of batch size.
	DEFAULT_METRIC_BUFFER_LIMIT = 10000
)

// RunningOutput contains the output configuration
type RunningOutput struct {
	Name              string
	Output            asgard.Output
	Config            *OutputConfig
	MetricBufferLimit int
	MetricBatchSize   int

	metrics     *buffer.Buffer
	failMetrics *buffer.Buffer

	// Guards against concurrent calls to the Output as described in #3009
	sync.Mutex
}

func NewRunningOutput(name string, output asgard.Output, batchSize int, bufferLimit int) *RunningOutput {
	if bufferLimit == 0 {
		bufferLimit = DEFAULT_METRIC_BUFFER_LIMIT
	}
	if batchSize == 0 {
		batchSize = DEFAULT_METRIC_BATCH_SIZE
	}
	ro := &RunningOutput{
		Name:              name,
		Output:            output,
		metrics:           buffer.NewBuffer(batchSize),
		failMetrics:       buffer.NewBuffer(bufferLimit),
		MetricBufferLimit: bufferLimit,
		MetricBatchSize:   batchSize,
	}
	return ro
}

// OutputConfig containing name and filter
type OutputConfig struct {
	Name   string
	Filter Filter
}

// AddMetric adds a metric to the output. This function can also write cached
// points if FlushBufferWhenFull is true.
func (ro *RunningOutput) AddMetric(m asgard.Metric) {
	if m == nil {
		return
	}

	// Filter any tagexclude/taginclude parameters before adding metric
	//if ro.Config.Filter.IsActive() {
	//	// In order to filter out tags, we need to create a new metric, since
	//	// metrics are immutable once created.
	//	name := m.Name()
	//	tags := m.Tags()
	//	fields := m.Fields()
	//	t := m.Time()
	//	if ok := ro.Config.Filter.Apply(name, fields, tags); !ok {
	//		return
	//	}
	//	// error is not possible if creating from another metric, so ignore.
	//	m, _ = metric.New(name, tags, fields, t)
	//}

	ro.metrics.Add(m)
	if ro.metrics.Len() == ro.MetricBatchSize {
		batch := ro.metrics.Batch(ro.MetricBatchSize)
		fmt.Println(batch)
		err := ro.write(batch)
		if err != nil {
			ro.failMetrics.Add(batch...)
		}
	}
}

// Write writes all cached points to this output.
func (ro *RunningOutput) Write() error {
	nFails, nMetrics := ro.failMetrics.Len(), ro.metrics.Len()
	log.Printf("D! Output [%s] buffer fullness: %d / %d metrics. ",
		ro.Name, nFails+nMetrics, ro.MetricBufferLimit)
	var err error
	if !ro.failMetrics.IsEmpty() {
		// how many batches of failed writes we need to write.
		nBatches := nFails/ro.MetricBatchSize + 1
		batchSize := ro.MetricBatchSize

		for i := 0; i < nBatches; i++ {
			// If it's the last batch, only grab the metrics that have not had
			// a write attempt already (this is primarily to preserve order).
			if i == nBatches-1 {
				batchSize = nFails % ro.MetricBatchSize
			}
			batch := ro.failMetrics.Batch(batchSize)
			// If we've already failed previous writes, don't bother trying to
			// write to this output again. We are not exiting the loop just so
			// that we can rotate the metrics to preserve order.
			if err == nil {
				err = ro.write(batch)
			}
			if err != nil {
				ro.failMetrics.Add(batch...)
			}
		}
	}

	batch := ro.metrics.Batch(ro.MetricBatchSize)
	// see comment above about not trying to write to an already failed output.
	// if ro.failMetrics is empty then err will always be nil at this point.
	if err == nil {
		err = ro.write(batch)
	}

	if err != nil {
		ro.failMetrics.Add(batch...)
		return err
	}
	return nil
}

func (ro *RunningOutput) write(metrics []asgard.Metric) error {
	nMetrics := len(metrics)
	if nMetrics == 0 {
		return nil
	}
	ro.Lock()
	defer ro.Unlock()
	start := time.Now()
	err := ro.Output.Write(metrics)
	elapsed := time.Since(start)
	if err == nil {
		log.Printf("D! Output [%s] wrote batch of %d metrics in %s\n",
			ro.Name, nMetrics, elapsed)
	}
	return err
}