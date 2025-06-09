package monitor

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/process"
)

type ProcessMonitor struct {
	stopChan chan struct{}
	Alerts   chan Alert
	Stats    chan ProcessStats
}

type ProcessStats struct {
	TotalProcesses int
	CPUUsage       float64
	MemoryUsage    float64
	Timestamp      time.Time
}

func NewProcessMonitor() *ProcessMonitor {
	return &ProcessMonitor{
		stopChan: make(chan struct{}),
		Alerts:   make(chan Alert, 100),
		Stats:    make(chan ProcessStats, 100),
	}
}

func (pm *ProcessMonitor) Start() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-pm.stopChan:
			return
		case <-ticker.C:
			pm.checkProcesses()
		}
	}
}

func (pm *ProcessMonitor) Stop() {
	close(pm.stopChan)
}

func (pm *ProcessMonitor) checkProcesses() {
	processes, err := process.Processes()
	if err != nil {
		log.Printf("Error getting processes: %v", err)
		return
	}

	totalCPU := 0.0
	totalMemory := 0.0
	processCount := 0

	for _, p := range processes {
		// Get process name
		name, err := p.Name()
		if err != nil {
			continue
		}

		processCount++

		// Check for suspicious processes
		if isSuspiciousProcess(name) {
			pm.Alerts <- Alert{
				Severity: "High",
				Message:  fmt.Sprintf("Suspicious process detected: %s", name),
				Time:     time.Now(),
			}
		}

		// Check process CPU usage
		cpuPercent, err := p.CPUPercent()
		if err == nil {
			totalCPU += cpuPercent
			if cpuPercent > 80 {
				pm.Alerts <- Alert{
					Severity: "Medium",
					Message:  fmt.Sprintf("High CPU usage detected for process %s: %.1f%%", name, cpuPercent),
					Time:     time.Now(),
				}
			}
		}

		// Check memory usage
		memInfo, err := p.MemoryInfo()
		if err == nil {
			totalMemory += float64(memInfo.RSS) / 1024 / 1024 // Convert to MB
		}
	}
	// Send stats update (normalize CPU usage by number of cores)
	numCPU := float64(runtime.NumCPU())
	normalizedCPU := totalCPU / numCPU

	pm.Stats <- ProcessStats{
		TotalProcesses: processCount,
		CPUUsage:       normalizedCPU,
		MemoryUsage:    totalMemory,
		Timestamp:      time.Now(),
	}
}

func isSuspiciousProcess(name string) bool {
	// Add your suspicious process detection logic here
	suspiciousProcesses := map[string]bool{
		"nc.exe":     true,
		"netcat.exe": true,
		"nmap.exe":   true,
	}
	return suspiciousProcesses[name]
}
