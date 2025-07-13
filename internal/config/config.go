package config

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"api_spray/pkg/types"
)

// ParseFlags parses command line flags and returns a Config
func ParseFlags() *types.Config {
	config := &types.Config{}

	// Parse flags
	flag.StringVar(&config.TargetsFile, "targets", "", "File containing target domains (required)")
	flag.StringVar(&config.Wordlist, "wordlist", "", "Wordlist file (required)")
	flag.StringVar(&config.Mode, "mode", "wildcards", "Scan mode: wildcards, directories, subdomains")
	flag.IntVar(&config.Threads, "threads", 50, "Number of concurrent threads")
	flag.IntVar(&config.Batch, "batch", 10, "Number of words per batch")
	flag.DurationVar(&config.Timeout, "timeout", 10*time.Second, "HTTP timeout")
	flag.StringVar(&config.OutDir, "outdir", "results", "Output directory")
	flag.BoolVar(&config.DisableHTTP, "disable-http", false, "Disable HTTP fallback")
	flag.BoolVar(&config.Resume, "resume", false, "Resume previous scan")
	flag.IntVar(&config.MaxRetries, "retries", 1, "Maximum number of retries per request")
	flag.StringVar(&config.UserAgent, "user-agent", "Mozilla/5.0 (compatible; api_spray/1.0)", "User agent string")
	flag.BoolVar(&config.FollowRedirs, "follow-redirects", true, "Follow HTTP redirects")

	var statusCodes string
	flag.StringVar(&statusCodes, "status-codes", "200", "Comma-separated list of success status codes")

	flag.Parse()

	// Validate required arguments
	if config.TargetsFile == "" || config.Wordlist == "" {
		fmt.Fprintf(os.Stderr, "Usage: %s -targets <file> -wordlist <file> [options]\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Parse status codes
	for _, code := range strings.Split(statusCodes, ",") {
		if c, err := strconv.Atoi(strings.TrimSpace(code)); err == nil {
			config.StatusCodes = append(config.StatusCodes, c)
		}
	}
	if len(config.StatusCodes) == 0 {
		config.StatusCodes = []int{200}
	}

	return config
}

// LoadLines loads lines from a file, filtering out empty lines and comments
func LoadLines(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}

	return lines, scanner.Err()
}
