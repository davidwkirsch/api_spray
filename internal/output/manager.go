package output

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"api_spray/pkg/types"
)

// Manager handles all output operations
type Manager struct {
	csvWriter  *csv.Writer
	csvFile    *os.File
	logFile    *os.File
	writeMutex sync.Mutex
	outDir     string
}

// NewManager creates a new output manager
func NewManager(outDir string) *Manager {
	return &Manager{
		outDir: outDir,
	}
}

// Initialize initializes output files and writers
func (om *Manager) Initialize() error {
	// Create output directory
	if err := os.MkdirAll(om.outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	csvPath := fmt.Sprintf("%s/results.csv", om.outDir)
	logPath := fmt.Sprintf("%s/scan.log", om.outDir)

	// Check if files exist for resume
	csvExists := false
	if _, err := os.Stat(csvPath); err == nil {
		csvExists = true
	}

	// Open CSV file
	var err error
	if csvExists {
		om.csvFile, err = os.OpenFile(csvPath, os.O_APPEND|os.O_WRONLY, 0644)
	} else {
		om.csvFile, err = os.Create(csvPath)
	}
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %w", err)
	}

	om.csvWriter = csv.NewWriter(om.csvFile)

	// Write CSV header if new file
	if !csvExists {
		header := []string{"target", "word", "url", "status_code", "content_length", "response_time_ms", "title", "error"}
		if err := om.csvWriter.Write(header); err != nil {
			return fmt.Errorf("failed to write CSV header: %w", err)
		}
		om.csvWriter.Flush()
	}

	// Open log file
	om.logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	return nil
}

// WriteResult writes a result to CSV and log files
func (om *Manager) WriteResult(result types.Result) error {
	om.writeMutex.Lock()
	defer om.writeMutex.Unlock()

	// Write to CSV
	record := []string{
		result.Target,
		result.Word,
		result.URL,
		strconv.Itoa(result.StatusCode),
		strconv.FormatInt(result.ContentLength, 10),
		strconv.FormatInt(result.ResponseTime, 10),
		result.Title,
		result.Error,
	}

	if err := om.csvWriter.Write(record); err != nil {
		return err
	}
	om.csvWriter.Flush()

	// Write to log if successful
	if result.StatusCode > 0 && result.Error == "" {
		logEntry := fmt.Sprintf("[%s] %s [%d] [%d] %dms\n",
			time.Now().Format("15:04:05"),
			result.URL,
			result.StatusCode,
			result.ContentLength,
			result.ResponseTime,
		)
		om.logFile.WriteString(logEntry)
	}

	return nil
}

// Close closes all file handles
func (om *Manager) Close() error {
	var errs []error

	if om.csvWriter != nil {
		om.csvWriter.Flush()
	}
	if om.csvFile != nil {
		if err := om.csvFile.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if om.logFile != nil {
		if err := om.logFile.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing files: %v", errs)
	}
	return nil
}

// ShouldSaveError determines if an error should be saved
func ShouldSaveError(errorMsg string) bool {
	// Don't save common DNS/network errors that indicate the subdomain doesn't exist
	dnsErrors := []string{
		"no such host",
		"server misbehaving",
		"connection refused",
		"network is unreachable",
		"host is down",
	}

	lowerError := strings.ToLower(errorMsg)
	for _, dnsErr := range dnsErrors {
		if strings.Contains(lowerError, dnsErr) {
			return false
		}
	}

	// Save other types of errors (timeouts, SSL errors, etc.)
	return true
}
