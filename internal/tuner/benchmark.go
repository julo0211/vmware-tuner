package tuner

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// BenchmarkTuner handles network benchmarking
type BenchmarkTuner struct{}

// NewBenchmarkTuner creates a new benchmark tuner
func NewBenchmarkTuner() *BenchmarkTuner {
	return &BenchmarkTuner{}
}

// Run performs the benchmark
func (bt *BenchmarkTuner) Run() error {
	PrintStep("Network Benchmark")

	// 1. Latency Test (Ping Gateway)
	PrintInfo("Testing latency...")
	gateway, err := getGateway()
	if err != nil {
		PrintWarning("Could not detect gateway: %v", err)
	} else {
		PrintInfo("Pinging gateway (%s)...", gateway)
		// ping -c 4 -i 0.2 <gateway>
		cmd := exec.Command("ping", "-c", "4", "-i", "0.2", gateway)
		output, err := cmd.CombinedOutput()
		if err != nil {
			PrintWarning("Ping failed: %v", err)
		} else {
			// Extract avg
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.Contains(line, "rtt") || strings.Contains(line, "avg") {
					fmt.Printf("  -> %s\n", strings.TrimSpace(line))
				}
			}
		}
	}

	// 2. Download Speed Test
	fmt.Println()
	PrintInfo("Testing download speed...")
	PrintInfo("Downloading 100MB test file (will be deleted immediately)...")

	url := "http://speedtest.tele2.net/100MB.zip" // Reliable public speedtest file
	tmpFile := "/tmp/vmware-tuner-speedtest.tmp"

	// START TIMER
	start := time.Now()

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %v", err)
	}
	defer resp.Body.Close()

	out, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	
	// CRITICAL: Ensure file is deleted
	defer func() {
		out.Close()
		os.Remove(tmpFile)
		PrintSuccess("Temporary file deleted")
	}()

	// Copy content
	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("download interrupted: %v", err)
	}

	// STOP TIMER
	elapsed := time.Since(start)

	// Calculate speed
	// written is bytes
	// elapsed is duration
	mb := float64(written) / 1024 / 1024
	seconds := elapsed.Seconds()
	speed := mb / seconds // MB/s

	fmt.Printf("  -> Downloaded %.2f MB in %.2f seconds\n", mb, seconds)
	PrintSuccess("Speed: %.2f MB/s (%.2f Mbps)", speed, speed*8)

	return nil
}

func getGateway() (string, error) {
	// ip route | grep default
	cmd := exec.Command("ip", "route")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "default") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				return parts[2], nil
			}
		}
	}
	return "", fmt.Errorf("no default route found")
}
