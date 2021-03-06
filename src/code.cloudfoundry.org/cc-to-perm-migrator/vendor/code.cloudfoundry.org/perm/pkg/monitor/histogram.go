package monitor

import (
	"sync"

	"time"

	"github.com/codahale/hdrhistogram"
)

type ThreadSafeHistogram struct {
	rw        *sync.RWMutex
	histogram *hdrhistogram.WindowedHistogram
}

func NewThreadSafeHistogram(windowSize int, sigfigs int) *ThreadSafeHistogram {
	h := hdrhistogram.NewWindowed(windowSize, 0, int64(time.Minute*10), sigfigs)

	return &ThreadSafeHistogram{
		rw:        &sync.RWMutex{},
		histogram: h,
	}
}

func (h *ThreadSafeHistogram) Max() int64 {
	h.rw.RLock()
	defer h.rw.RUnlock()

	return h.histogram.Current.Max()
}

func (h *ThreadSafeHistogram) RecordValue(v int64) error {
	h.rw.Lock()
	defer h.rw.Unlock()

	return h.histogram.Current.RecordValue(v)
}

func (h *ThreadSafeHistogram) ValueAtQuantile(q float64) int64 {
	h.rw.RLock()
	defer h.rw.RUnlock()

	return h.histogram.Merge().ValueAtQuantile(q)
}

func (h *ThreadSafeHistogram) Rotate() {
	h.rw.Lock()
	defer h.rw.Unlock()

	h.histogram.Rotate()
}
