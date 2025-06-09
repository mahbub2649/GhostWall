package dashboard

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"time"

	"ghostwall/internal/monitor"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type Dashboard struct {
	networkMonitor *monitor.NetworkMonitor
	fileMonitor    *monitor.FileMonitor
	processMonitor *monitor.ProcessMonitor
	upgrader       websocket.Upgrader
	clients        map[*websocket.Conn]bool
}

func NewDashboard(nm *monitor.NetworkMonitor, fm *monitor.FileMonitor, pm *monitor.ProcessMonitor) *Dashboard {
	return &Dashboard{
		networkMonitor: nm,
		fileMonitor:    fm,
		processMonitor: pm,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for local development
			},
		},
		clients: make(map[*websocket.Conn]bool),
	}
}

func (d *Dashboard) Start() {
	r := mux.NewRouter()

	// Serve static files
	fs := http.FileServer(http.Dir("web/static"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	// WebSocket endpoint
	r.HandleFunc("/ws", d.handleWebSocket)

	// Main dashboard page
	r.HandleFunc("/", d.handleDashboard)

	// API endpoints
	r.HandleFunc("/api/alerts", d.handleAlerts).Methods("GET")
	r.HandleFunc("/api/status", d.handleStatus).Methods("GET")

	log.Println("Dashboard starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func (d *Dashboard) handleDashboard(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("web/templates/dashboard.html"))
	tmpl.Execute(w, nil)
}

func (d *Dashboard) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := d.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading to WebSocket: %v", err)
		return
	}
	defer conn.Close()

	d.clients[conn] = true
	defer delete(d.clients, conn)

	// Keep the connection alive and send updates
	for {
		// Send system status updates
		status := d.getSystemStatus()
		if err := conn.WriteJSON(status); err != nil {
			break
		}
		time.Sleep(5 * time.Second)
	}
}

func (d *Dashboard) handleAlerts(w http.ResponseWriter, r *http.Request) {
	// Get alerts from all monitors
	alerts := []monitor.Alert{}

	// Add alerts from network monitor
	select {
	case alert := <-d.networkMonitor.Alerts:
		alerts = append(alerts, alert)
	default:
	}

	// Add alerts from file monitor
	select {
	case alert := <-d.fileMonitor.Alerts:
		alerts = append(alerts, alert)
	default:
	}

	// Add alerts from process monitor
	select {
	case alert := <-d.processMonitor.Alerts:
		alerts = append(alerts, alert)
	default:
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

func (d *Dashboard) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := d.getSystemStatus()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (d *Dashboard) getSystemStatus() map[string]interface{} {
	// Collect latest stats from monitors
	networkStats := d.getLatestNetworkStats()
	processStats := d.getLatestProcessStats()
	fileStats := d.getLatestFileStats()

	return map[string]interface{}{
		"timestamp": time.Now(),
		"status":    "active",
		"network":   networkStats,
		"processes": processStats,
		"files":     fileStats,
	}
}

func (d *Dashboard) getLatestNetworkStats() map[string]interface{} {
	select {
	case stats := <-d.networkMonitor.Stats:
		return map[string]interface{}{
			"connections": stats.TotalConnections,
			"bytesSent":   stats.BytesSent,
			"bytesRecv":   stats.BytesReceived,
			"activePorts": len(stats.ActivePorts),
		}
	default:
		return map[string]interface{}{
			"connections": 0,
			"bytesSent":   0,
			"bytesRecv":   0,
			"activePorts": 0,
		}
	}
}

func (d *Dashboard) getLatestProcessStats() map[string]interface{} {
	select {
	case stats := <-d.processMonitor.Stats:
		return map[string]interface{}{
			"total":    stats.TotalProcesses,
			"cpuUsage": stats.CPUUsage,
			"memUsage": stats.MemoryUsage,
		}
	default:
		return map[string]interface{}{
			"total":    0,
			"cpuUsage": 0,
			"memUsage": 0,
		}
	}
}

func (d *Dashboard) getLatestFileStats() map[string]interface{} {
	select {
	case stats := <-d.fileMonitor.Stats:
		return map[string]interface{}{
			"totalEvents":    stats.TotalEvents,
			"recentChanges":  stats.RecentChanges,
			"monitoredPaths": stats.MonitoredPaths,
		}
	default:
		return map[string]interface{}{
			"totalEvents":    0,
			"recentChanges":  []string{},
			"monitoredPaths": 0,
		}
	}
}
