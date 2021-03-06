package main

import (
	"sync"
	"time"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/perm/cmd"
	"code.cloudfoundry.org/perm/pkg/monitor"
)

const (
	ProbeHistogramWindow      = 5 // Minutes
	ProbeHistogramRefreshTime = 1 * time.Minute
)

func RunProbeWithFrequency(
	logger lager.Logger,
	probe *monitor.Probe,
	statter monitor.PermStatter,
	frequency time.Duration,
	requestDuration time.Duration,
	timeout time.Duration,
) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		for range time.NewTicker(ProbeHistogramRefreshTime).C {
			statter.Rotate()
		}
	}()

	go func() {
		defer wg.Done()

		for range time.NewTicker(frequency).C {
			cmd.RecordProbeResults(logger, probe, statter, requestDuration, timeout)
		}
	}()

	wg.Wait()
}
