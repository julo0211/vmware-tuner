package tuner

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// UpdateTuner handles system updates
type UpdateTuner struct {
	Distro *DistroManager
}

// NewUpdateTuner creates a new update tuner
func NewUpdateTuner(distro *DistroManager) *UpdateTuner {
	return &UpdateTuner{
		Distro: distro,
	}
}

// Run performs the update
func (ut *UpdateTuner) Run(hasInternet bool) error {
	PrintStep("Safe System Update")

	if !hasInternet {
		PrintWarning("Mode Hors-Ligne activé : Pas de mises à jour système possibles.")
		return fmt.Errorf("offline mode")
	}

	// 1. Check Disk Space
	PrintInfo("Checking disk space...")
	cmd := exec.Command("df", "--output=avail", "/")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check disk space: %w", err)
	}

	// Simple check: ensure we have enough bytes (rough approximation from output)
	// Real implementation would parse strictly. Here we rely on the user seeing the output if we just show it?
	// No, let's try to be safer.
	// Output is like:
	// Avail
	// 10240000
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) >= 2 {
		availStr := strings.TrimSpace(lines[1])
		var avail int64
		fmt.Sscanf(availStr, "%d", &avail)

		// df outputs 1K blocks usually
		availMB := avail / 1024
		if availMB < 1000 {
			PrintError("Insufficient disk space! Only %d MB free.", availMB)
			PrintInfo("At least 1000 MB is recommended for safe updates.")
			return fmt.Errorf("disk space check failed")
		}
		PrintSuccess("Disk space OK (%d MB free)", availMB)
	}

	// 2. Run Update
	fmt.Println()
	PrintInfo("Ready to update system packages.")
	fmt.Print("Continue? (y/n): ")
	var resp string
	fmt.Scanln(&resp)
	if resp != "y" {
		PrintInfo("Cancelled")
		return nil
	}

	var updateCmd *exec.Cmd
	if ut.Distro.Type == DistroDebian {
		// Interactive update
		cmdStr := "apt-get update && apt-get upgrade"
		updateCmd = exec.Command("bash", "-c", cmdStr)
	} else if ut.Distro.Type == DistroRHEL {
		if _, err := exec.LookPath("dnf"); err == nil {
			updateCmd = exec.Command("dnf", "update")
		} else {
			updateCmd = exec.Command("yum", "update")
		}
	} else {
		return fmt.Errorf("unsupported distribution for auto-update")
	}

	updateCmd.Stdout = os.Stdout
	updateCmd.Stderr = os.Stderr
	updateCmd.Stdin = os.Stdin

	if err := updateCmd.Run(); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	PrintSuccess("System updated successfully!")

	// 3. Check Reboot
	rebootNeeded := false
	if ut.Distro.Type == DistroDebian {
		if _, err := os.Stat("/var/run/reboot-required"); err == nil {
			rebootNeeded = true
		}
	} else if ut.Distro.Type == DistroRHEL {
		// needs-restarting -r (yum-utils)
		if err := exec.Command("needs-restarting", "-r").Run(); err != nil {
			// Exit code 1 means reboot needed
			rebootNeeded = true
		}
	}

	if rebootNeeded {
		PrintWarning("A reboot is required to apply updates.")
		fmt.Print("Reboot now? (y/n): ")
		fmt.Scanln(&resp)
		if resp == "y" {
			exec.Command("reboot").Run()
		}
	} else {
		PrintSuccess("No reboot required.")
	}

	return nil
}
