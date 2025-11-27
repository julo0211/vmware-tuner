package main

import (
	"os/exec"
	"strings"
)

// HardwareTuner handles hardware verification
type HardwareTuner struct {
	Distro *DistroManager
}

// NewHardwareTuner creates a new hardware tuner
func NewHardwareTuner(distro *DistroManager) *HardwareTuner {
	return &HardwareTuner{
		Distro: distro,
	}
}

// Run performs the hardware check
func (ht *HardwareTuner) Run() error {
	PrintStep("Virtual Hardware Inspector")

	// 1. Check Network Adapter Type
	PrintInfo("Checking Network Adapter...")
	// Get interface name
	cmd := exec.Command("ip", "-o", "link", "show")
	out, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		foundVmxnet3 := false
		for _, line := range lines {
			if strings.Contains(line, "link/ether") {
				parts := strings.Fields(line)
				if len(parts) > 1 {
					iface := strings.Trim(parts[1], ":")
					// Check driver
					driverCmd := exec.Command("ethtool", "-i", iface)
					driverOut, _ := driverCmd.Output()
					if strings.Contains(string(driverOut), "driver: vmxnet3") {
						foundVmxnet3 = true
						PrintSuccess("Interface %s is using vmxnet3 driver", iface)
					} else if strings.Contains(string(driverOut), "driver: e1000") {
						PrintWarning("Interface %s is using legacy e1000 driver (Upgrade to vmxnet3 recommended)", iface)
					}
				}
			}
		}
		if !foundVmxnet3 {
			PrintInfo("No vmxnet3 adapters found (or ethtool missing)")
		}
	}

	// 2. Check SCSI Controller
	PrintInfo("Checking SCSI Controller...")
	// lspci is best, but might not be installed.
	// Try installing pciutils if missing? No, read-only check shouldn't install stuff ideally.
	// Let's try to detect via sysfs or dmesg
	
	// Check for vmw_pvscsi or nvme module
	if out, err := exec.Command("lsmod").Output(); err == nil {
		output := string(out)
		if strings.Contains(output, "vmw_pvscsi") {
			PrintSuccess("VMware Paravirtual SCSI (PVSCSI) driver loaded")
		} else if strings.Contains(output, "nvme") {
			PrintSuccess("NVMe Controller detected (High Performance)")
		} else if strings.Contains(output, "mptspi") || strings.Contains(output, "mptsas") {
			PrintInfo("Detected LSI Logic Controller (Standard)")
			PrintInfo("Recommendation: Upgrade to VMware Paravirtual (PVSCSI) for better I/O performance")
		} else {
			// Check if it's built-in or just not used
			PrintWarning("Optimal Storage Controller not found (PVSCSI/NVMe)")
		}
	}

	// 3. Check 3D Acceleration (often unnecessary on servers)
	// Hard to check from guest without logs, skip for now.

	return nil
}
