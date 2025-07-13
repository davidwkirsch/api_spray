package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"api_spray/internal/config"
	"api_spray/internal/scanner"
)

func main() {
	// Parse command line flags
	cfg := config.ParseFlags()

	// Load input files
	targets, err := config.LoadLines(cfg.TargetsFile)
	if err != nil {
		log.Fatalf("Failed to load targets: %v", err)
	}

	wordlist, err := config.LoadLines(cfg.Wordlist)
	if err != nil {
		log.Fatalf("Failed to load wordlist: %v", err)
	}

	// Create scanner
	scan, err := scanner.NewScanner(cfg)
	if err != nil {
		log.Fatalf("Failed to create scanner: %v", err)
	}
	defer scan.Close()

	// Initialize scanner
	if err := scan.Initialize(); err != nil {
		log.Fatalf("Failed to initialize scanner: %v", err)
	}

	// Handle resume functionality
	if cfg.Resume {
		// Load existing progress and results
		if err := scan.LoadProgress(); err != nil {
			log.Printf("Warning: %v", err)
		}

		if err := scan.LoadCompletedWork(); err != nil {
			log.Printf("Warning: %v", err)
		}
	}

	// Print banner
	fmt.Printf("\n=== Go API Spray Scanner ===\n")
	fmt.Printf("Mode: %s\n", strings.ToUpper(cfg.Mode))
	fmt.Printf("Targets: %d | Words: %d | Threads: %d | Batch: %d\n",
		len(targets), len(wordlist), cfg.Threads, cfg.Batch)
	fmt.Printf("Timeout: %v | Status Codes: %v\n", cfg.Timeout, cfg.StatusCodes)
	if cfg.DisableHTTP {
		fmt.Println("HTTP fallback: DISABLED")
	}
	fmt.Printf("Started: %s\n\n", time.Now().Format("15:04:05"))

	// Run scan
	if err := scan.Run(targets, wordlist); err != nil {
		log.Fatalf("Scan failed: %v", err)
	}

	// Print final statistics
	total, success, errors, timeouts, filtered := scan.GetStats()
	fmt.Printf("Scan completed: %s\n", time.Now().Format("15:04:05"))
	fmt.Printf("Final stats: %d total, %d success, %d errors, %d timeouts, %d filtered\n",
		total, success, errors, timeouts, filtered)
	fmt.Printf("Results saved in: %s\n", cfg.OutDir)
}
