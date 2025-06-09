package main

import (
	"ghostwall/internal/dashboard"
	"ghostwall/internal/monitor"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Initialize monitors
	networkMonitor := monitor.NewNetworkMonitor()
	fileMonitor := monitor.NewFileMonitor()
	processMonitor := monitor.NewProcessMonitor()

	// Start monitoring
	go networkMonitor.Start()
	go fileMonitor.Start()
	go processMonitor.Start()

	// Initialize and start dashboard
	dashboard := dashboard.NewDashboard(networkMonitor, fileMonitor, processMonitor)
	go dashboard.Start()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Cleanup
	networkMonitor.Stop()
	fileMonitor.Stop()
	processMonitor.Stop()
	log.Println("GhostWall shutting down...")
}
