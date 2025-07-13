package types

import "time"

// Config holds all scanner configuration
type Config struct {
	TargetsFile  string
	Wordlist     string
	Mode         string
	Threads      int
	Batch        int
	Timeout      time.Duration
	OutDir       string
	DisableHTTP  bool
	Resume       bool
	MaxRetries   int
	UserAgent    string
	FollowRedirs bool
	StatusCodes  []int
}

// ScanMode represents different scanning modes
type ScanMode int

const (
	ModeWildcards ScanMode = iota
	ModeDirectories
	ModeSubdomains
)

// GetMode returns the scan mode enum
func (c *Config) GetMode() ScanMode {
	switch c.Mode {
	case "directories":
		return ModeDirectories
	case "subdomains":
		return ModeSubdomains
	default:
		return ModeWildcards
	}
}

// Result represents a scan result
type Result struct {
	Target        string `json:"target" csv:"target"`
	Word          string `json:"word" csv:"word"`
	URL           string `json:"url" csv:"url"`
	StatusCode    int    `json:"status_code" csv:"status_code"`
	ContentLength int64  `json:"content_length" csv:"content_length"`
	ResponseTime  int64  `json:"response_time_ms" csv:"response_time_ms"`
	Title         string `json:"title,omitempty" csv:"title"`
	Error         string `json:"error,omitempty" csv:"error"`
}

// FalsePositiveTracker tracks response sizes that appear to be false positives
type FalsePositiveTracker struct {
	// Map of target -> status_code -> response_size -> count
	SizeTracking map[string]map[int]map[int64]int `json:"size_tracking"`
	// Map of target -> status_code -> set of filtered sizes
	FilteredSizes map[string]map[int]map[int64]bool `json:"filtered_sizes"`
	// Minimum threshold to consider a size as false positive
	Threshold int `json:"threshold"`
}

// Progress tracks scan progress for resume functionality
type Progress struct {
	LastBatch            int                   `json:"last_batch"`
	TotalBatches         int                   `json:"total_batches"`
	CompletedCount       int                   `json:"completed_count"`
	TotalWork            int                   `json:"total_work"`
	Timestamp            time.Time             `json:"timestamp"`
	StartTime            time.Time             `json:"start_time"`
	LastSaveTime         time.Time             `json:"last_save_time"`
	FalsePositiveTracker *FalsePositiveTracker `json:"false_positive_tracker"`
}

// NewFalsePositiveTracker creates a new false positive tracker
func NewFalsePositiveTracker() *FalsePositiveTracker {
	return &FalsePositiveTracker{
		SizeTracking:  make(map[string]map[int]map[int64]int),
		FilteredSizes: make(map[string]map[int]map[int64]bool),
		Threshold:     10,
	}
}

// TrackResponseSize tracks response sizes for false positive detection
func (fp *FalsePositiveTracker) TrackResponseSize(target string, statusCode int, contentLength int64) {
	if fp.SizeTracking[target] == nil {
		fp.SizeTracking[target] = make(map[int]map[int64]int)
	}
	if fp.SizeTracking[target][statusCode] == nil {
		fp.SizeTracking[target][statusCode] = make(map[int64]int)
	}

	fp.SizeTracking[target][statusCode][contentLength]++

	// Check if this size should be filtered
	if fp.SizeTracking[target][statusCode][contentLength] >= fp.Threshold {
		if fp.FilteredSizes[target] == nil {
			fp.FilteredSizes[target] = make(map[int]map[int64]bool)
		}
		if fp.FilteredSizes[target][statusCode] == nil {
			fp.FilteredSizes[target][statusCode] = make(map[int64]bool)
		}
		fp.FilteredSizes[target][statusCode][contentLength] = true
	}
}

// ShouldFilter returns true if the response should be filtered as false positive
func (fp *FalsePositiveTracker) ShouldFilter(target string, statusCode int, contentLength int64) bool {
	if fp.FilteredSizes[target] == nil {
		return false
	}
	if fp.FilteredSizes[target][statusCode] == nil {
		return false
	}
	return fp.FilteredSizes[target][statusCode][contentLength]
}
