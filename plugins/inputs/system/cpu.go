package system

import (
	"fmt"
	"github.com/anabiozz/asgard"
	"github.com/anabiozz/asgard/plugins/inputs"
	"time"

	"github.com/shirou/gopsutil/cpu"
)

// CPUStats ...
type CPUStats struct {
	ps        PS
	lastStats map[string]cpu.TimesStat

	PerCPU         bool `toml:"percpu"`
	TotalCPU       bool `toml:"totalcpu"`
	CollectCPUTime bool `toml:"collect_cpu_time"`
	ReportActive   bool `toml:"report_active"`
}

func (_ *CPUStats) Description() string {
	return "Read metrics about memory usage"
}

func (_ *CPUStats) SampleConfig() string { return "" }

// Gather ...
func (s *CPUStats) Gather(acc asgard.Accumulator) error {
	times, err := s.ps.CPUTimes(s.PerCPU, s.TotalCPU)
	if err != nil {
		return fmt.Errorf("error getting CPU info: %s", err)
	}
	now := time.Now()

	for _, cts := range times {
		tags := map[string]string{
			"cpu": cts.CPU,
		}

		total := totalCpuTime(cts)
		active := activeCpuTime(cts)

		 if s.CollectCPUTime {
		 	// Add cpu time metrics
		 	fieldsC := map[string]interface{}{
		 		"time_user":       cts.User,
		 		"time_system":     cts.System,
		 		"time_idle":       cts.Idle,
		 		"time_nice":       cts.Nice,
		 		"time_iowait":     cts.Iowait,
		 		"time_irq":        cts.Irq,
		 		"time_softirq":    cts.Softirq,
		 		"time_steal":      cts.Steal,
		 		"time_guest":      cts.Guest,
		 		"time_guest_nice": cts.GuestNice,
		 	}
		 	if s.ReportActive {
		 		fieldsC["time_active"] = activeCpuTime(cts)
		 	}
		 	acc.AddCounter("cpu", fieldsC, tags, now)
		 }

		// Add in percentage
		if len(s.lastStats) == 0 {
			// If it's the 1st gather, can't get CPU Usage stats yet
			continue
		}

		lastCts, ok := s.lastStats[cts.CPU]
		if !ok {
			continue
		}
		lastTotal := totalCpuTime(lastCts)
		lastActive := activeCpuTime(lastCts)
		totalDelta := total - lastTotal

		if totalDelta < 0 {
			err = fmt.Errorf("Error: current total CPU time is less than previous total CPU time")
			break
		}

		if totalDelta == 0 {
			continue
		}

		fieldsG := map[string]interface{}{
			"usage_user":       100 * (cts.User - lastCts.User - (cts.Guest - lastCts.Guest)) / totalDelta,
			"usage_system":     100 * (cts.System - lastCts.System) / totalDelta,
			"usage_idle":       100 * (cts.Idle - lastCts.Idle) / totalDelta,
			"usage_nice":       100 * (cts.Nice - lastCts.Nice - (cts.GuestNice - lastCts.GuestNice)) / totalDelta,
			"usage_iowait":     100 * (cts.Iowait - lastCts.Iowait) / totalDelta,
			"usage_irq":        100 * (cts.Irq - lastCts.Irq) / totalDelta,
			"usage_softirq":    100 * (cts.Softirq - lastCts.Softirq) / totalDelta,
			"usage_steal":      100 * (cts.Steal - lastCts.Steal) / totalDelta,
			"usage_guest":      100 * (cts.Guest - lastCts.Guest) / totalDelta,
			"usage_guest_nice": 100 * (cts.GuestNice - lastCts.GuestNice) / totalDelta,
		}
		if s.ReportActive {
			fieldsG["usage_active"] = 100 * (active - lastActive) / totalDelta
		}
		acc.AddGauge("cpu", fieldsG, tags, now)
	}

	s.lastStats = make(map[string]cpu.TimesStat)
	for _, cts := range times {
		s.lastStats[cts.CPU] = cts
	}
	return err
}

// totalCpuTime ...
func totalCpuTime(t cpu.TimesStat) float64 {
	total := t.User + t.System + t.Nice + t.Iowait + t.Irq + t.Softirq + t.Steal + t.Idle
	return total
}

// activeCpuTime ...
func activeCpuTime(t cpu.TimesStat) float64 {
	active := totalCpuTime(t) - t.Idle
	return active
}

func init() {
	inputs.Add("cpu", func() asgard.Input {
		return &CPUStats{
			// PerCPU:         false,
			TotalCPU: true,
			// CollectCPUTime: false,
			ps: newSystemPS(),
		}
	})
}
