package tuner

import (
	"fmt"
	"os/exec"
)

// TimeSyncTuner handles time synchronization
type TimeSyncTuner struct {
	Distro *DistroManager
}

// NewTimeSyncTuner creates a new time sync tuner
func NewTimeSyncTuner(distro *DistroManager) *TimeSyncTuner {
	return &TimeSyncTuner{
		Distro: distro,
	}
}

// Run performs the time sync check and fix
func (t *TimeSyncTuner) Run(hasInternet bool) error {
	PrintStep("Time Synchronization Doctor")

	// 1. Check for existing NTP services
	services := []string{"chronyd", "ntp", "systemd-timesyncd"}
	activeService := ""

	for _, svc := range services {
		cmd := exec.Command("systemctl", "is-active", svc)
		if err := cmd.Run(); err == nil {
			activeService = svc
			break
		}
	}

	if activeService != "" {
		PrintSuccess("Time synchronization is active via: %s", activeService)

		// Force sync
		PrintInfo("Forcing time synchronization...")
		if activeService == "chronyd" {
			exec.Command("chronyc", "makestep").Run()
		} else if activeService == "systemd-timesyncd" {
			// systemd-timesyncd doesn't have a simple force command, restart triggers it
			exec.Command("systemctl", "restart", "systemd-timesyncd").Run()
		}

		// Ensure VMware Tools sync is disabled to avoid conflict
		PrintInfo("Disabling VMware Tools periodic time sync (best practice with NTP)...")
		exec.Command("vmware-toolbox-cmd", "timesync", "disable").Run()

		return nil
	}

	PrintWarning("No active NTP service found!")

	// 2. If no NTP, offer to enable VMware Tools sync or install chrony
	fmt.Println()
	fmt.Println("Options:")

	if hasInternet {
		fmt.Println("  [1] Install/Enable Chrony (Recommended)")
	} else {
		// Greyed out or hidden
		PrintInfo("  [1] Install Chrony (Unavailable - Offline)")
	}
	fmt.Println("  [2] Enable VMware Tools Host Sync (Fallback)")
	fmt.Println("  [3] Skip")
	fmt.Print("Choice: ")

	var choice string
	fmt.Scanln(&choice)

	if choice == "1" {
		if !hasInternet {
			PrintWarning("Cannot install Chrony in offline mode. Please use VMware Tools Sync.")
			return nil
		}
		pkg := "chrony"
		if t.Distro.Type == DistroDebian {
			pkg = "chrony"
		}
		if err := t.Distro.InstallPackage(pkg); err != nil {
			return err
		}
		exec.Command("systemctl", "enable", "--now", "chronyd").Run()
		exec.Command("chronyc", "makestep").Run()
		PrintSuccess("Chrony installed and synchronized")
	} else if choice == "2" {
		if err := exec.Command("vmware-toolbox-cmd", "timesync", "enable").Run(); err != nil {
			return fmt.Errorf("failed to enable vmtools sync: %v", err)
		}
		PrintSuccess("VMware Tools Host Sync enabled")
	} else {
		PrintInfo("Skipping time sync")
	}

	return nil
}
