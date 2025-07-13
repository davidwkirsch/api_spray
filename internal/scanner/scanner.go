package scanner

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"api_spray/internal/http"
	"api_spray/internal/output"
	"api_spray/internal/progress"
	"api_spray/pkg/types"
)

// Scanner is the main scanning engine
type Scanner struct {
	config      *types.Config
	httpClient  *http.Client
	progressMgr *progress.Manager
	outputMgr   *output.Manager
	stats       *Statistics
}

// Statistics tracks scan statistics
type Statistics struct {
	totalRequests int64
	successCount  int64
	errorCount    int64
	timeoutCount  int64
	filteredCount int64
}

// NewScanner creates a new scanner instance
func NewScanner(config *types.Config) (*Scanner, error) {
	return &Scanner{
		config:      config,
		httpClient:  http.NewClient(config),
		progressMgr: progress.NewManager(config.OutDir),
		outputMgr:   output.NewManager(config.OutDir),
		stats:       &Statistics{},
	}, nil
}

// Initialize initializes the scanner
func (s *Scanner) Initialize() error {
	return s.outputMgr.Initialize()
}

// LoadProgress loads existing progress for resume functionality
func (s *Scanner) LoadProgress() error {
	return s.progressMgr.LoadProgress()
}

// LoadCompletedWork loads already completed target/word combinations
func (s *Scanner) LoadCompletedWork() error {
	return s.progressMgr.LoadCompletedWork(s.config.OutDir)
}

// SaveProgress saves current progress
func (s *Scanner) SaveProgress() error {
	return s.progressMgr.SaveProgress()
}

// Close closes all file handles
func (s *Scanner) Close() error {
	return s.outputMgr.Close()
}

// GetStats returns current statistics
func (s *Scanner) GetStats() (total, success, errors, timeouts, filtered int64) {
	return atomic.LoadInt64(&s.stats.totalRequests),
		atomic.LoadInt64(&s.stats.successCount),
		atomic.LoadInt64(&s.stats.errorCount),
		atomic.LoadInt64(&s.stats.timeoutCount),
		atomic.LoadInt64(&s.stats.filteredCount)
}

// UpdateStats updates internal statistics
func (s *Scanner) UpdateStats(statType string, count int64) {
	switch statType {
	case "total":
		atomic.AddInt64(&s.stats.totalRequests, count)
	case "success":
		atomic.AddInt64(&s.stats.successCount, count)
	case "error":
		atomic.AddInt64(&s.stats.errorCount, count)
	case "timeout":
		atomic.AddInt64(&s.stats.timeoutCount, count)
	case "filtered":
		atomic.AddInt64(&s.stats.filteredCount, count)
	}
}

// TestURL tests a single URL and returns the result
func (s *Scanner) TestURL(ctx context.Context, target, word, url string) types.Result {
	s.UpdateStats("total", 1)

	result := http.TestURL(ctx, s.httpClient, target, word, url, s.config.StatusCodes, s.config.DisableHTTP)

	// Categorize errors for statistics
	if result.Error != "" {
		if strings.Contains(result.Error, "timeout") {
			s.UpdateStats("timeout", 1)
		} else if strings.Contains(strings.ToLower(result.Error), "no such host") {
			// Don't count DNS errors in main error stats
		} else {
			s.UpdateStats("error", 1)
		}
		return result
	}

	// Check if status code is in allowed list
	allowed := false
	for _, code := range s.config.StatusCodes {
		if result.StatusCode == code {
			allowed = true
			break
		}
	}

	if allowed {
		// Track response size for false positive detection
		s.progressMgr.TrackResponseSize(target, result.StatusCode, result.ContentLength)
		s.UpdateStats("success", 1)
	} else {
		s.UpdateStats("error", 1)
	}

	return result
}

// Run executes the main scanning logic
func (s *Scanner) Run(targets, wordlist []string) error {
	// Initialize progress tracking only if not already loaded
	progress := s.progressMgr.GetProgress()
	if progress == nil {
		progress = &types.Progress{
			TotalBatches:         (len(wordlist) + s.config.Batch - 1) / s.config.Batch,
			TotalWork:            len(targets) * len(wordlist),
			StartTime:            time.Now(),
			FalsePositiveTracker: types.NewFalsePositiveTracker(),
		}
		s.progressMgr.SetProgress(progress)
	} else {
		// Update values that might have changed
		progress.TotalBatches = (len(wordlist) + s.config.Batch - 1) / s.config.Batch
		progress.TotalWork = len(targets) * len(wordlist)
		if progress.FalsePositiveTracker == nil {
			progress.FalsePositiveTracker = types.NewFalsePositiveTracker()
		}
	}

	completedCount := s.progressMgr.CountCompleted()
	startBatch := progress.LastBatch

	fmt.Printf("Resume status: %d/%d items completed (%.1f%%)\n",
		completedCount, progress.TotalWork,
		float64(completedCount)/float64(progress.TotalWork)*100)

	if completedCount == progress.TotalWork {
		fmt.Println("Scan already completed!")
		return nil
	}

	fmt.Printf("Starting from batch %d/%d\n", startBatch+1, progress.TotalBatches)

	// Process in batches
	for batchNum := startBatch; batchNum < progress.TotalBatches; batchNum++ {
		startIdx := batchNum * s.config.Batch
		endIdx := startIdx + s.config.Batch
		if endIdx > len(wordlist) {
			endIdx = len(wordlist)
		}

		wordBatch := wordlist[startIdx:endIdx]

		fmt.Printf("── Batch %d/%d (Words %d-%d) %s\n",
			batchNum+1, progress.TotalBatches,
			startIdx+1, endIdx,
			time.Now().Format("15:04:05"))

		if err := s.processBatch(targets, wordBatch); err != nil {
			return fmt.Errorf("error processing batch %d: %w", batchNum, err)
		}

		// Update and save progress
		progress.LastBatch = batchNum + 1
		progress.CompletedCount = s.progressMgr.CountCompleted()
		if err := s.SaveProgress(); err != nil {
			log.Printf("Warning: failed to save progress: %v", err)
		}

		// Print statistics
		total, success, errors, timeouts, filtered := s.GetStats()
		fmt.Printf("   Stats: %d total, %d success, %d errors, %d timeouts, %d filtered\n\n",
			total, success, errors, timeouts, filtered)
	}

	// Clean up progress file on completion
	s.progressMgr.CleanupProgressFile()
	fmt.Println("Scan completed successfully!")

	return nil
}

// processBatch processes a batch of words against all targets
func (s *Scanner) processBatch(targets, words []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create work channel
	work := make(chan struct {
		target, word string
	}, s.config.Threads*2)

	// Count total work and completed work for this batch
	totalWork := len(targets) * len(words)
	completedWork := 0

	// Pre-check completed work for this batch
	for _, target := range targets {
		for _, word := range words {
			if s.progressMgr.IsCompleted(target, word) {
				completedWork++
			}
		}
	}

	fmt.Printf("   Batch progress: %d/%d already completed\n", completedWork, totalWork)

	// If all work is completed, skip this batch
	if completedWork == totalWork {
		fmt.Printf("   Batch already completed, skipping...\n")
		return nil
	}

	// Start workers
	var wg sync.WaitGroup
	processedCount := int64(0)

	for i := 0; i < s.config.Threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range work {
				// Skip if already completed
				if s.progressMgr.IsCompleted(job.target, job.word) {
					continue
				}

				url := http.GenerateURL(job.target, job.word, s.config.GetMode())
				result := s.TestURL(ctx, job.target, job.word, url)

				// Check if this should be filtered as false positive
				shouldFilter := false
				if result.StatusCode > 0 {
					shouldFilter = s.progressMgr.ShouldFilter(job.target, result.StatusCode, result.ContentLength)
					if shouldFilter {
						s.UpdateStats("filtered", 1)
					}
				}

				// Determine if we should save this result
				shouldSave := false

				if result.StatusCode > 0 && !shouldFilter {
					// Got an HTTP response and it's not filtered - check if it's a status code we care about
					for _, code := range s.config.StatusCodes {
						if result.StatusCode == code {
							shouldSave = true
							break
						}
					}
				} else if result.Error != "" {
					// Only save certain types of errors (not DNS failures)
					shouldSave = output.ShouldSaveError(result.Error)
				}

				if shouldSave {
					if err := s.outputMgr.WriteResult(result); err != nil {
						log.Printf("Error writing result: %v", err)
					}
				}

				// Always mark as completed (even DNS failures and filtered results)
				s.progressMgr.MarkCompleted(job.target, job.word)

				// Update processed count
				atomic.AddInt64(&processedCount, 1)

				// Periodic progress update
				if processedCount%100 == 0 {
					fmt.Printf("   Processed: %d/%d\n", processedCount, totalWork-completedWork)
				}
			}
		}()
	}

	// Send work
	go func() {
		defer close(work)
		for _, target := range targets {
			for _, word := range words {
				// Only send work that hasn't been completed
				if !s.progressMgr.IsCompleted(target, word) {
					select {
					case work <- struct {
						target, word string
					}{target, word}:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	// Wait for completion
	wg.Wait()

	fmt.Printf("   Batch completed: %d new requests processed\n", processedCount)
	return nil
}
