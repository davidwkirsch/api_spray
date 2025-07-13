package progress

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"

	"api_spray/pkg/types"
)

// Manager handles scan progress tracking and persistence
type Manager struct {
	progress     *types.Progress
	progressFile string
	completed    sync.Map
	fpTracker    *types.FalsePositiveTracker
	fpMutex      sync.RWMutex
}

// NewManager creates a new progress manager
func NewManager(outDir string) *Manager {
	return &Manager{
		progressFile: fmt.Sprintf("%s/scan_progress.json", outDir),
		fpTracker:    types.NewFalsePositiveTracker(),
	}
}

// LoadProgress loads existing progress for resume functionality
func (pm *Manager) LoadProgress() error {
	data, err := os.ReadFile(pm.progressFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no previous scan found")
		}
		return fmt.Errorf("failed to read progress file: %w", err)
	}

	pm.progress = &types.Progress{}
	if err := json.Unmarshal(data, pm.progress); err != nil {
		return fmt.Errorf("failed to parse progress file: %w", err)
	}

	// Initialize false positive tracker if not present
	if pm.progress.FalsePositiveTracker == nil {
		pm.progress.FalsePositiveTracker = types.NewFalsePositiveTracker()
	}
	pm.fpTracker = pm.progress.FalsePositiveTracker

	return nil
}

// SaveProgress saves current progress
func (pm *Manager) SaveProgress() error {
	if pm.progress == nil {
		return fmt.Errorf("no progress to save")
	}

	pm.progress.FalsePositiveTracker = pm.fpTracker
	data, err := json.MarshalIndent(pm.progress, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal progress: %w", err)
	}

	return os.WriteFile(pm.progressFile, data, 0644)
}

// LoadCompletedWork loads already completed target/word combinations
func (pm *Manager) LoadCompletedWork(outDir string) error {
	csvPath := fmt.Sprintf("%s/results.csv", outDir)
	file, err := os.Open(csvPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no previous results found")
		}
		return fmt.Errorf("failed to open results file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // Allow variable number of fields

	// Skip header
	if _, err := reader.Read(); err != nil {
		return fmt.Errorf("failed to read CSV header: %w", err)
	}

	count := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		if len(record) >= 2 {
			key := fmt.Sprintf("%s|%s", record[0], record[1]) // target|word
			pm.completed.Store(key, true)
			count++

			// Track response sizes for false positive detection if we have the data
			if len(record) >= 5 {
				if statusCode, err := strconv.Atoi(record[3]); err == nil && statusCode > 0 {
					if contentLength, err := strconv.ParseInt(record[4], 10, 64); err == nil {
						pm.fpTracker.TrackResponseSize(record[0], statusCode, contentLength)
					}
				}
			}
		}
	}

	fmt.Printf("Loaded %d completed work items from previous scan\n", count)
	return nil
}

// GetProgress returns the current progress
func (pm *Manager) GetProgress() *types.Progress {
	return pm.progress
}

// SetProgress sets the progress
func (pm *Manager) SetProgress(progress *types.Progress) {
	pm.progress = progress
	if progress.FalsePositiveTracker != nil {
		pm.fpTracker = progress.FalsePositiveTracker
	}
}

// IsCompleted checks if a target/word combination is already completed
func (pm *Manager) IsCompleted(target, word string) bool {
	key := fmt.Sprintf("%s|%s", target, word)
	_, exists := pm.completed.Load(key)
	return exists
}

// MarkCompleted marks a target/word combination as completed
func (pm *Manager) MarkCompleted(target, word string) {
	key := fmt.Sprintf("%s|%s", target, word)
	pm.completed.Store(key, true)
}

// CountCompleted counts completed work items
func (pm *Manager) CountCompleted() int {
	count := 0
	pm.completed.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// TrackResponseSize tracks response size for false positive detection
func (pm *Manager) TrackResponseSize(target string, statusCode int, contentLength int64) {
	pm.fpMutex.Lock()
	defer pm.fpMutex.Unlock()
	pm.fpTracker.TrackResponseSize(target, statusCode, contentLength)
}

// ShouldFilter checks if response should be filtered as false positive
func (pm *Manager) ShouldFilter(target string, statusCode int, contentLength int64) bool {
	pm.fpMutex.RLock()
	defer pm.fpMutex.RUnlock()
	return pm.fpTracker.ShouldFilter(target, statusCode, contentLength)
}

// CleanupProgressFile removes the progress file on completion
func (pm *Manager) CleanupProgressFile() error {
	return os.Remove(pm.progressFile)
}
