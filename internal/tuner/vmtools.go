package tuner

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// VMToolsTuner handles open-vm-tools installation and configuration
type VMToolsTuner struct {
	Distro *DistroManager
	DryRun bool
}

// NewVMToolsTuner creates a new VM tools tuner
func NewVMToolsTuner(dryRun bool, distro *DistroManager) *VMToolsTuner {
	return &VMToolsTuner{
		Distro: distro,
		DryRun: dryRun,
	}
}

// CheckInstalled checks if open-vm-tools is installed
func (vt *VMToolsTuner) CheckInstalled() bool {
	_, err := exec.LookPath("vmtoolsd")
	return err == nil
}

// Apply installs and enables open-vm-tools
func (vt *VMToolsTuner) Apply() error {
	PrintStep("Checking VMware Tools")

	if vt.CheckInstalled() {
		PrintSuccess("open-vm-tools is already installed")
		return vt.ensureService()
	}

	PrintInfo("open-vm-tools is missing")
	
	if vt.DryRun {
		PrintInfo("Would install open-vm-tools package")
		return nil
	}

	// Install package
	if err := vt.Distro.InstallPackage("open-vm-tools"); err != nil {
		return fmt.Errorf("failed to install open-vm-tools: %w", err)
	}

	return vt.ensureService()
}

// ensureService makes sure the service is running
func (vt *VMToolsTuner) ensureService() error {
	if vt.DryRun {
		return nil
	}

	// Service name is usually open-vm-tools or vmtoolsd
	serviceName := "open-vm-tools"
	if vt.Distro.Type == DistroRHEL {
		// On RHEL/CentOS it might be vmtoolsd
		serviceName = "vmtoolsd"
	}

	PrintInfo("Ensuring %s service is running...", serviceName)
	
	// Enable
	exec.Command("systemctl", "enable", serviceName).Run()
	
	// Start
	cmd := exec.Command("systemctl", "start", serviceName)
	if err := cmd.Run(); err != nil {
		// Try alternative name if failed
		if serviceName == "open-vm-tools" {
			serviceName = "vmtoolsd"
		} else {
			serviceName = "open-vm-tools"
		}
		exec.Command("systemctl", "enable", serviceName).Run()
		exec.Command("systemctl", "start", serviceName).Run()
	}

	PrintSuccess("VMware Tools service configured")
	return nil
}

// CheckUpdateStatus returns installed, updateAvailable, daysSinceLastUpdate, error
func (vt *VMToolsTuner) CheckUpdateStatus() (bool, bool, int, error) {
	if !vt.CheckInstalled() {
		return false, false, 0, nil
	}

	// Check binary age
	binPath, err := exec.LookPath("vmtoolsd")
	days := 0
	if err == nil {
		info, err := os.Stat(binPath)
		if err == nil {
			days = int(time.Since(info.ModTime()).Hours() / 24)
		}
	}

	// Check for updates
	updateAvailable := vt.IsUpdateAvailable()

	return true, updateAvailable, days, nil
}

// IsUpdateAvailable checks if an update is available via package manager
func (vt *VMToolsTuner) IsUpdateAvailable() bool {
	// This is a "best effort" check based on local cache
	if vt.Distro.Type == DistroDebian {
		// apt-get -s install --only-upgrade open-vm-tools
		cmd := exec.Command("apt-get", "-s", "install", "--only-upgrade", "open-vm-tools")
		out, err := cmd.CombinedOutput()
		if err == nil && strings.Contains(string(out), "Inst open-vm-tools") {
			return true
		}
	} else if vt.Distro.Type == DistroRHEL {
		// yum check-update open-vm-tools
		// Exit code 100 means updates available
		cmd := exec.Command("yum", "check-update", "open-vm-tools")
		err := cmd.Run()
		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				if exitError.ExitCode() == 100 {
					return true
				}
			}
		}
	}
	return false
}

// UpdateTools attempts to update the package
func (vt *VMToolsTuner) UpdateTools() error {
	PrintInfo("Updating open-vm-tools...")
	if err := vt.Distro.InstallPackage("open-vm-tools"); err != nil {
		return err
	}
	return vt.ensureService()
}
