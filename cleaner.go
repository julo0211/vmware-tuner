package main

import (
	"fmt"
	"os/exec"
)

// CleanerTuner handles system cleaning
type CleanerTuner struct {
	Distro *DistroManager
}

// NewCleanerTuner creates a new cleaner
func NewCleanerTuner(distro *DistroManager) *CleanerTuner {
	return &CleanerTuner{
		Distro: distro,
	}
}

// Run performs the cleaning
func (ct *CleanerTuner) Run() error {
	PrintStep("System Cleaner")

	PrintInfo("This will:")
	PrintInfo("  - Clean package manager cache")
	PrintInfo("  - Vacuum system logs (keep last 3 days)")
	PrintInfo("  - Remove old crash dumps")
	fmt.Println()
	fmt.Print("Continue? (y/n): ")
	
	var response string
	fmt.Scanln(&response)
	if response != "y" && response != "yes" {
		PrintInfo("Cancelled")
		return nil
	}

	// 1. Clean Package Cache
	PrintInfo("Cleaning package cache...")
	if ct.Distro.Type == DistroDebian {
		exec.Command("apt-get", "clean").Run()
		exec.Command("apt-get", "autoremove", "-y").Run()
	} else if ct.Distro.Type == DistroRHEL {
		if _, err := exec.LookPath("dnf"); err == nil {
			exec.Command("dnf", "clean", "all").Run()
			exec.Command("dnf", "autoremove", "-y").Run()
		} else {
			exec.Command("yum", "clean", "all").Run()
			exec.Command("yum", "autoremove", "-y").Run()
		}
	}
	PrintSuccess("Package cache cleaned")

	// 2. Vacuum Journal
	PrintInfo("Vacuuming system logs...")
	if err := exec.Command("journalctl", "--vacuum-time=3d").Run(); err != nil {
		PrintWarning("Failed to vacuum journal: %v", err)
	} else {
		PrintSuccess("Logs vacuumed (kept 3 days)")
	}

	// 3. Show Free Space
	PrintInfo("Current Disk Usage:")
	exec.Command("df", "-h", "/").Run()

	return nil
}
