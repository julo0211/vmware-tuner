package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// SwapTuner handles swap management
type SwapTuner struct{}

// NewSwapTuner creates a new swap tuner
func NewSwapTuner() *SwapTuner {
	return &SwapTuner{}
}

// Run performs the swap check and creation
func (st *SwapTuner) Run() error {
	PrintStep("Swap Manager")

	// 1. Check current swap
	cmd := exec.Command("swapon", "--show")
	out, err := cmd.Output()
	if err == nil && len(out) > 0 {
		PrintSuccess("Swap is currently active:")
		fmt.Println(string(out))
		return nil
	}

	PrintWarning("No active swap detected!")
	PrintInfo("Running without swap can cause the OOM Killer to crash applications.")
	fmt.Println()
	fmt.Print("Create a 2GB swapfile? (y/n): ")
	
	var response string
	fmt.Scanln(&response)
	if response != "y" && response != "yes" {
		PrintInfo("Cancelled")
		return nil
	}

	swapFile := "/swapfile"

	// 2. Create file
	PrintInfo("Creating 2GB swapfile at %s...", swapFile)
	// Try fallocate first (fast)
	if err := exec.Command("fallocate", "-l", "2G", swapFile).Run(); err != nil {
		PrintInfo("fallocate failed, trying dd...")
		// dd if=/dev/zero of=/swapfile bs=1M count=2048
		if err := exec.Command("dd", "if=/dev/zero", "of="+swapFile, "bs=1M", "count=2048").Run(); err != nil {
			return fmt.Errorf("failed to create swapfile: %w", err)
		}
	}

	// 3. Permissions
	os.Chmod(swapFile, 0600)

	// 4. Mkswap
	PrintInfo("Formatting swap...")
	if err := exec.Command("mkswap", swapFile).Run(); err != nil {
		return fmt.Errorf("mkswap failed: %w", err)
	}

	// 5. Swapon
	PrintInfo("Activating swap...")
	if err := exec.Command("swapon", swapFile).Run(); err != nil {
		return fmt.Errorf("swapon failed: %w", err)
	}

	// 6. Persist in fstab
	PrintInfo("Updating /etc/fstab...")
	fstabEntry := fmt.Sprintf("%s none swap sw 0 0\n", swapFile)
	
	// Read fstab to check if already exists
	content, _ := os.ReadFile("/etc/fstab")
	if !strings.Contains(string(content), swapFile) {
		f, err := os.OpenFile("/etc/fstab", os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			PrintWarning("Failed to open fstab: %v", err)
		} else {
			defer f.Close()
			if _, err := f.WriteString(fstabEntry); err != nil {
				PrintWarning("Failed to write to fstab: %v", err)
			} else {
				PrintSuccess("Added to /etc/fstab")
			}
		}
	}

	PrintSuccess("Swap created successfully!")
	return nil
}
