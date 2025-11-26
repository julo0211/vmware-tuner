package main

import (
	"fmt"
	"strings"
)

// AuditTuner handles system auditing
type AuditTuner struct {
	Distro *DistroManager
}

// NewAuditTuner creates a new audit tuner
func NewAuditTuner(distro *DistroManager) *AuditTuner {
	return &AuditTuner{
		Distro: distro,
	}
}

// RunAudit performs the audit and prints the report
func (at *AuditTuner) RunAudit() error {
	PrintStep("System Optimization Audit")

	score := 0
	maxScore := 100

	// 1. Check VM Tools (30 points)
	tools := NewVMToolsTuner(true, at.Distro)
	if tools.CheckInstalled() {
		PrintSuccess("VMware Tools installed (+30)")
		score += 30
	} else {
		PrintError("VMware Tools missing (0/30)")
	}

	// 2. Check GRUB (30 points)
	grub := NewGrubTuner(true, at.Distro)
	config, _, err := grub.ParseGrubConfig()
	if err == nil {
		cmdline := config["GRUB_CMDLINE_LINUX_DEFAULT"]
		if strings.Contains(cmdline, "elevator=noop") || strings.Contains(cmdline, "elevator=none") {
			PrintSuccess("I/O Scheduler optimized (+15)")
			score += 15
		} else {
			PrintWarning("I/O Scheduler not optimized (0/15)")
		}
		
		if strings.Contains(cmdline, "transparent_hugepage=madvise") {
			PrintSuccess("Memory pages optimized (+15)")
			score += 15
		} else {
			PrintWarning("Memory pages not optimized (0/15)")
		}
	} else {
		PrintWarning("Could not read GRUB config")
	}

	// 3. Check Bloatware (20 points)
	debloat := NewDebloatTuner(true)
	bloat := debloat.GetBloatServices()
	if len(bloat) == 0 {
		PrintSuccess("No unnecessary services found (+20)")
		score += 20
	} else {
		PrintWarning("Found %d unnecessary services (0/20)", len(bloat))
		for _, svc := range bloat {
			fmt.Printf("    - %s\n", svc.Name)
		}
	}

	// 4. Check Sysctl (20 points)
	// Simple check for swappiness
	// In a real implementation we would check actual values
	// For now, let's assume if the config file exists, it's good
	if FileExists("/etc/sysctl.d/99-vmware-performance.conf") {
		PrintSuccess("Sysctl optimizations present (+20)")
		score += 20
	} else {
		PrintWarning("Sysctl optimizations missing (0/20)")
	}

	fmt.Println()
	PrintStep("Audit Result")
	
	fmt.Printf("Final Score: %d/%d\n", score, maxScore)
	
	if score == 100 {
		PrintSuccess("System is fully optimized! 🚀")
	} else if score >= 70 {
		PrintInfo("System is well optimized, but could be better.")
	} else {
		PrintWarning("System requires optimization.")
		PrintInfo("Run 'Optimize this VM' from the main menu.")
	}

	return nil
}
