package monitor

import (
	"fmt"
	"log"
	"time"

	"github.com/shirou/gopsutil/net"
)

type NetworkMonitor struct {
	stopChan chan struct{}
	Alerts   chan Alert
	Stats    chan NetworkStats
}

type NetworkStats struct {
	TotalConnections int
	ActivePorts      []uint32
	BytesSent        uint64
	BytesReceived    uint64
	Timestamp        time.Time
}

type Alert struct {
	Severity string
	Message  string
	Time     time.Time
}

func NewNetworkMonitor() *NetworkMonitor {
	return &NetworkMonitor{
		stopChan: make(chan struct{}),
		Alerts:   make(chan Alert, 100),
		Stats:    make(chan NetworkStats, 100),
	}
}

func (nm *NetworkMonitor) Start() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-nm.stopChan:
			return
		case <-ticker.C:
			nm.checkConnections()
		}
	}
}

func (nm *NetworkMonitor) Stop() {
	close(nm.stopChan)
}

func (nm *NetworkMonitor) checkConnections() {
	connections, err := net.Connections("all")
	if err != nil {
		log.Printf("Error getting network connections: %v", err)
		return
	}

	activePorts := make([]uint32, 0)
	connectionCount := 0

	for _, conn := range connections {
		connectionCount++

		// Track active ports
		if conn.Laddr.Port != 0 {
			activePorts = append(activePorts, conn.Laddr.Port)
		}

		// Check for suspicious ports
		if isSuspiciousPort(conn.Laddr.Port) {
			nm.Alerts <- Alert{
				Severity: "High",
				Message:  fmt.Sprintf("Suspicious port detected: %d", conn.Laddr.Port),
				Time:     time.Now(),
			}
		}

		// Check for suspicious IPs
		if conn.Raddr.IP != "" && isSuspiciousIP(conn.Raddr.IP) {
			nm.Alerts <- Alert{
				Severity: "High",
				Message:  fmt.Sprintf("Suspicious IP detected: %s", conn.Raddr.IP),
				Time:     time.Now(),
			}
		}
	}

	// Get network IO stats
	ioStats, err := net.IOCounters(false)
	var bytesSent, bytesReceived uint64
	if err == nil && len(ioStats) > 0 {
		bytesSent = ioStats[0].BytesSent
		bytesReceived = ioStats[0].BytesRecv
	}

	// Send stats update
	nm.Stats <- NetworkStats{
		TotalConnections: connectionCount,
		ActivePorts:      activePorts,
		BytesSent:        bytesSent,
		BytesReceived:    bytesReceived,
		Timestamp:        time.Now(),
	}
}

func isSuspiciousPort(port uint32) bool {
	// Add your suspicious port detection logic here
	suspiciousPorts := map[uint32]bool{
		22:   true, // SSH
		23:   true, // Telnet
		3389: true, // RDP
	}
	return suspiciousPorts[port]
}

func isSuspiciousIP(ip string) bool {
	// Add your suspicious IP detection logic here
	// This is a placeholder implementation
	return false
}
