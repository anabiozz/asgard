package buffer

import (
	"sync"
	"github.com/anabiozz/asgard"
)

// Buffer is an object for storing metrics in a circular buffer.
type Buffer struct {
	buf chan asgard.Metric
	mu sync.Mutex
}

// NewBuffer returns a Buffer
//   size is the maximum number of metrics that Buffer will cache. If Add is
//   called when the buffer is full, then the oldest metric(s) will be dropped.
func NewBuffer(size int) *Buffer {
	return &Buffer{
		buf: make(chan asgard.Metric, size),
	}
}

// Len returns the current length of the buffer.
func (b *Buffer) Len() int {
	return len(b.buf)
}

// IsEmpty returns true if Buffer is empty.
func (b *Buffer) IsEmpty() bool {
	return len(b.buf) == 0
}

// Batch returns a batch of metrics of size batchSize.
// the batch will be of maximum length batchSize. It can be less than batchSize,
// if the length of Buffer is less than batchSize.
func (b *Buffer) Batch(batchSize int) []asgard.Metric {
	b.mu.Lock()
	n := min(len(b.buf), batchSize)
	out := make([]asgard.Metric, n)
	for i := 0; i < n; i++ {
		out[i] = <-b.buf
	}
	b.mu.Unlock()
	return out
}

func min(a, b int) int {
	if b < a {
		return b
	}
	return a
}

// Add adds metrics to the buffer.
func (b *Buffer) Add(metrics ...asgard.Metric) {
	for i, _ := range metrics {
		select {
		case b.buf <- metrics[i]:
		default:
			b.mu.Lock()
			<-b.buf
			b.buf <- metrics[i]
			b.mu.Unlock()
		}
	}
}