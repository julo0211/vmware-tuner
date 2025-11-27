package tuner

import (
	"fmt"
	"os/exec"
)

// DebloatTuner handles disabling unnecessary services
type DebloatTuner struct {
	DryRun bool
}

// NewDebloatTuner creates a new debloat tuner
func NewDebloatTuner(dryRun bool) *DebloatTuner {
	return &DebloatTuner{
		DryRun: dryRun,
	}
}

// Service represents a system service
type Service struct {
	Name        string
	Description string
	Active      bool
}

// GetBloatServices returns a list of potentially unnecessary services
func (dt *DebloatTuner) GetBloatServices() []Service {
	// List of services to check
	targets := []Service{
		{Name: "cups", Description: "Printing service (CUPS)"},
		{Name: "cups-browsed", Description: "Printer discovery"},
		{Name: "avahi-daemon", Description: "mDNS/DNS-SD (Avahi)"},
		{Name: "bluetooth", Description: "Bluetooth service"},
		{Name: "wpa_supplicant", Description: "Wi-Fi security (WPA)"},
		{Name: "modemmanager", Description: "Modem Manager"},
		{Name: "snapd", Description: "Snap Package Manager (consumes loop devices)"},
		{Name: "lxcfs", Description: "LXC File System (if not using containers)"},
		{Name: "multipathd", Description: "Multipath Device Daemon (unless using SAN)"},
	}

	var found []Service
	for _, svc := range targets {
		if dt.isServiceActive(svc.Name) {
			svc.Active = true
			found = append(found, svc)
		}
	}

	return found
}

// isServiceActive checks if a service is active
func (dt *DebloatTuner) isServiceActive(name string) bool {
	cmd := exec.Command("systemctl", "is-active", name)
	err := cmd.Run()
	return err == nil
}

// Apply disables the identified services
func (dt *DebloatTuner) Apply(backup *BackupManager) error {
	PrintStep("Checking for unnecessary services (Server Slim Mode)")

	services := dt.GetBloatServices()
	if len(services) == 0 {
		PrintSuccess("System is already clean (no bloatware found)")
		return nil
	}

	PrintInfo("Found %d unnecessary services:", len(services))
	for _, svc := range services {
		fmt.Printf("  - %s: %s\n", svc.Name, svc.Description)
	}

	if dt.DryRun {
		PrintInfo("Would disable these services")
		return nil
	}

	// Ask for confirmation if not already confirmed in main
	// For now, we assume the user opted-in via flag or interactive prompt in main

	// Backup services first
	var serviceNames []string
	for _, svc := range services {
		serviceNames = append(serviceNames, svc.Name)
	}
	if err := backup.BackupServices(serviceNames); err != nil {
		PrintWarning("Failed to backup service list: %v", err)
	}

	for _, svc := range services {
		PrintInfo("Disabling %s...", svc.Name)
		
		// Stop
		exec.Command("systemctl", "stop", svc.Name).Run()
		
		// Disable
		if err := exec.Command("systemctl", "disable", svc.Name).Run(); err != nil {
			PrintWarning("Failed to disable %s: %v", svc.Name, err)
		} else {
			PrintSuccess("Disabled %s", svc.Name)
		}
	}

	return nil
}

// DisableServices disables a specific list of services
func (dt *DebloatTuner) DisableServices(services []Service, backup *BackupManager) error {
	// Backup services first
	var serviceNames []string
	for _, svc := range services {
		serviceNames = append(serviceNames, svc.Name)
	}
	if err := backup.BackupServices(serviceNames); err != nil {
		PrintWarning("Failed to backup service list: %v", err)
	}

	for _, svc := range services {
		PrintInfo("Disabling %s...", svc.Name)
		
		if dt.DryRun {
			continue
		}
		
		// Stop
		exec.Command("systemctl", "stop", svc.Name).Run()
		
		// Disable
		if err := exec.Command("systemctl", "disable", svc.Name).Run(); err != nil {
			PrintWarning("Failed to disable %s: %v", svc.Name, err)
		} else {
			PrintSuccess("Disabled %s", svc.Name)
		}
	}
	return nil
}
