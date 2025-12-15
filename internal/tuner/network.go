package tuner

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// NetworkTuner handles network optimization
type NetworkTuner struct {
	ServicePath string
	DryRun      bool
}

// NewNetworkTuner creates a new network tuner
func NewNetworkTuner(dryRun bool) *NetworkTuner {
	return &NetworkTuner{
		ServicePath: "/etc/systemd/system/network-tuning.service",
		DryRun:      dryRun,
	}
}

// GetSystemdService returns the systemd service for network tuning
func (nt *NetworkTuner) GetSystemdService() string {
	return `[Unit]
Description=Network Performance Tuning for VMware
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
RemainOnExit=yes

# Increase ring buffers (ONLY for vmxnet3 to avoid e1000 hangs)
ExecStart=/bin/bash -c 'for iface in $(ls /sys/class/net/ | grep -E "^(ens|eth)"); do if ethtool -i $iface | grep -q "driver: vmxnet3"; then ethtool -G $iface rx 4096 tx 4096 2>/dev/null || true; fi; done'

# Enable hardware offloading features (ONLY for vmxnet3)
ExecStart=/bin/bash -c 'for iface in $(ls /sys/class/net/ | grep -E "^(ens|eth)"); do if ethtool -i $iface | grep -q "driver: vmxnet3"; then ethtool -K $iface gso on gro on tso on 2>/dev/null || true; fi; done'

# Set interrupt coalescing (ONLY for vmxnet3)
ExecStart=/bin/bash -c 'for iface in $(ls /sys/class/net/ | grep -E "^(ens|eth)"); do if ethtool -i $iface | grep -q "driver: vmxnet3"; then ethtool -C $iface rx-usecs 10 tx-usecs 10 2>/dev/null || true; fi; done'

[Install]
WantedBy=multi-user.target
`
}

// Apply applies network optimizations
func (nt *NetworkTuner) Apply(backup *BackupManager) error {
	PrintStep("Configuring network optimizations")

	service := nt.GetSystemdService()

	if nt.DryRun {
		PrintInfo("Would create: %s", nt.ServicePath)
		PrintInfo("Service file preview:")
		fmt.Println(service)
		return nil
	}

	// Backup existing service if it exists
	if err := backup.BackupFile(nt.ServicePath); err != nil {
		return fmt.Errorf("failed to backup network service: %w", err)
	}

	// Write systemd service
	if err := os.WriteFile(nt.ServicePath, []byte(service), 0644); err != nil {
		return fmt.Errorf("failed to write network service: %w", err)
	}

	PrintSuccess("Created %s", nt.ServicePath)

	// Reload systemd
	PrintInfo("Reloading systemd daemon...")
	cmd := exec.Command("systemctl", "daemon-reload")
	if output, err := cmd.CombinedOutput(); err != nil {
		PrintWarning("Failed to reload systemd: %v", err)
		fmt.Println(string(output))
	}

	// Enable the service
	PrintInfo("Enabling network tuning service...")
	cmd = exec.Command("systemctl", "enable", "network-tuning.service")
	if output, err := cmd.CombinedOutput(); err != nil {
		PrintWarning("Failed to enable service: %v", err)
		fmt.Println(string(output))
	}

	// Start the service (apply changes now)
	PrintInfo("Starting network tuning service...")
	cmd = exec.Command("systemctl", "start", "network-tuning.service")
	if output, err := cmd.CombinedOutput(); err != nil {
		PrintWarning("Failed to start service: %v", err)
		fmt.Println(string(output))
		PrintWarning("Network tuning will be applied on next boot")
	} else {
		PrintSuccess("Network tuning applied immediately")
	}

	return nil
}

// ShowCurrent displays current network settings
func (nt *NetworkTuner) ShowCurrent() error {
	PrintStep("Current network interface settings")

	// Get network interfaces
	interfaces, err := nt.getNetworkInterfaces()
	if err != nil {
		return err
	}

	for _, iface := range interfaces {
		fmt.Printf("\n  Interface: %s\n", iface)

		// Get ring buffer settings
		cmd := exec.Command("ethtool", "-g", iface)
		if output, err := cmd.Output(); err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.Contains(line, "Current") || strings.Contains(line, "RX") || strings.Contains(line, "TX") {
					fmt.Printf("    %s\n", strings.TrimSpace(line))
				}
			}
		}

		// Get offload features
		cmd = exec.Command("ethtool", "-k", iface)
		if output, err := cmd.Output(); err == nil {
			features := []string{"tcp-segmentation-offload", "generic-receive-offload", "generic-segmentation-offload"}
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				for _, feature := range features {
					if strings.Contains(line, feature+":") {
						fmt.Printf("    %s\n", strings.TrimSpace(line))
					}
				}
			}
		}
	}

	return nil
}

// getNetworkInterfaces returns a list of network interfaces
func (nt *NetworkTuner) getNetworkInterfaces() ([]string, error) {
	cmd := exec.Command("bash", "-c", "ls /sys/class/net/ | grep -E '^(ens|eth)'")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	interfaces := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(interfaces) == 1 && interfaces[0] == "" {
		return []string{}, nil
	}

	return interfaces, nil
}

// Verify checks if the network tuning service exists
func (nt *NetworkTuner) Verify() error {
	if _, err := os.Stat(nt.ServicePath); os.IsNotExist(err) {
		return fmt.Errorf("network tuning service not found: %s", nt.ServicePath)
	}

	PrintSuccess("Network tuning service exists")

	// Check if service is enabled
	cmd := exec.Command("systemctl", "is-enabled", "network-tuning.service")
	if output, err := cmd.Output(); err == nil {
		status := strings.TrimSpace(string(output))
		if status == "enabled" {
			PrintSuccess("Network tuning service is enabled")
		} else {
			PrintWarning("Network tuning service is not enabled")
		}
	}

	return nil
}

// CheckPacketDrops checks for packet drops on all interfaces using ethtool -S
func (nt *NetworkTuner) CheckPacketDrops() error {
	PrintStep("Checking for network packet drops")

	interfaces, err := nt.getNetworkInterfaces()
	if err != nil {
		return err
	}

	for _, iface := range interfaces {
		fmt.Printf("Interface: %s\n", iface)

		// Use RunCommandSilent from exec_utils (we need to export it or duplicate logic if not exported?
		// Actually I added RunCommandSilent to generic package, let's check if I can use it.
		// It is in the same package 'tuner', so yes.)
		output, err := RunCommandSilent("ethtool", "-S", iface)
		if err != nil {
			PrintWarning("  Could not get statistics: %v", err)
			continue
		}

		lines := strings.Split(output, "\n")
		dropsFound := false
		for _, line := range lines {
			// Look for drop or error keywords
			if strings.Contains(line, "drop") || strings.Contains(line, "error") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					// format usually: "rx_dropped: 123"
					valStr := parts[len(parts)-1]
					if valStr != "0" {
						PrintWarning("  %s", strings.TrimSpace(line))
						dropsFound = true
					}
				}
			}
		}

		if !dropsFound {
			PrintSuccess("  No packet drops or errors detected")
		}
	}
	return nil
}
