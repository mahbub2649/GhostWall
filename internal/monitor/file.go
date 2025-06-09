package monitor

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

type FileMonitor struct {
	watcher        *fsnotify.Watcher
	stopChan       chan struct{}
	Alerts         chan Alert
	Stats          chan FileStats
	paths          []string
	eventCount     int
	recentChanges  []FileEvent
	maxRecentItems int
}

type FileEvent struct {
	FileName  string
	Operation string
	Timestamp time.Time
}

type FileStats struct {
	TotalEvents    int
	RecentChanges  []FileEvent
	MonitoredPaths int
	Timestamp      time.Time
}

func NewFileMonitor() *FileMonitor {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("Error creating file watcher:", err)
	}
	return &FileMonitor{
		watcher:  watcher,
		stopChan: make(chan struct{}),
		Alerts:   make(chan Alert, 100),
		Stats:    make(chan FileStats, 100),
		paths: []string{
			filepath.Join(os.Getenv("USERPROFILE"), "Documents"),
			filepath.Join(os.Getenv("USERPROFILE"), "Downloads"),
			"C:\\Windows\\System32", // Monitor system directory
		},
		eventCount:     0,
		recentChanges:  make([]FileEvent, 0),
		maxRecentItems: 50, // Keep last 50 events
	}
}

func (fm *FileMonitor) Start() {
	// Start watching the specified paths
	for _, path := range fm.paths {
		err := fm.watcher.Add(path)
		if err != nil {
			log.Printf("Error watching path %s: %v", path, err)
		}
	}

	// Send periodic stats updates
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-fm.stopChan:
			return
		case <-ticker.C:
			fm.sendStats()
		case event := <-fm.watcher.Events:
			fm.handleFileEvent(event)
		case err := <-fm.watcher.Errors:
			log.Printf("File watcher error: %v", err)
		}
	}
}

func (fm *FileMonitor) Stop() {
	close(fm.stopChan)
	fm.watcher.Close()
}

func (fm *FileMonitor) handleFileEvent(event fsnotify.Event) {
	// Update file statistics first
	fm.eventCount++

	// Add to recent changes with timestamp and operation
	fileEvent := FileEvent{
		FileName:  filepath.Base(event.Name),
		Operation: event.Op.String(),
		Timestamp: time.Now(),
	}

	fm.recentChanges = append(fm.recentChanges, fileEvent)

	// Keep only the most recent items
	if len(fm.recentChanges) > fm.maxRecentItems {
		fm.recentChanges = fm.recentChanges[len(fm.recentChanges)-fm.maxRecentItems:]
	}

	// Determine the severity based on the event type
	severity := "Low"
	if event.Op&fsnotify.Write == fsnotify.Write {
		severity = "Medium"
	}
	if event.Op&fsnotify.Create == fsnotify.Create {
		severity = "High"
	}

	// Create an alert for the file event
	fm.Alerts <- Alert{
		Severity: severity,
		Message:  fmt.Sprintf("File event: %s on %s", event.Op, filepath.Base(event.Name)),
		Time:     time.Now(),
	}
}

func (fm *FileMonitor) sendStats() {
	fm.Stats <- FileStats{
		TotalEvents:    fm.eventCount,
		RecentChanges:  fm.recentChanges,
		MonitoredPaths: len(fm.paths),
		Timestamp:      time.Now(),
	}
}
