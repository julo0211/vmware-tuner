package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// DiskTuner handles disk expansion
type DiskTuner struct {
	Distro *DistroManager
}

// NewDiskTuner creates a new disk tuner
func NewDiskTuner(distro *DistroManager) *DiskTuner {
	return &DiskTuner{
		Distro: distro,
	}
}

// ExpandRoot expands the root partition and filesystem
func (dt *DiskTuner) ExpandRoot() error {
	PrintStep("Disk Expansion Assistant")

	PrintWarning("⚠️  WARNING: Disk operations carry a risk of data loss.")
	PrintWarning("Please ensure you have a snapshot or backup before proceeding.")
	fmt.Println()
	fmt.Print("Do you want to continue? (yes/no): ")
	var response string
	fmt.Scanln(&response)
	if response != "yes" {
		PrintInfo("Operation cancelled")
		return nil
	}

	// 1. Install cloud-guest-utils (contains growpart) if missing
	if err := dt.Distro.InstallPackage("cloud-guest-utils"); err != nil {
		// Try cloud-utils on some distros
		dt.Distro.InstallPackage("cloud-utils")
	}

	// 2. Identify root device
	// This is a simplified approach. In production code, we'd parse lsblk -J
	// For now, we assume standard /dev/sda or /dev/nvme0n1 layout
	
	PrintInfo("Analyzing disk structure...")
	
	// Detect root partition
	cmd := exec.Command("findmnt", "/", "-o", "SOURCE", "-n")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to detect root device: %w", err)
	}
	rootPart := strings.TrimSpace(string(output))
	PrintInfo("Root partition: %s", rootPart)

	// Detect disk and partition number
	// e.g. /dev/sda2 -> disk=/dev/sda, part=2
	var disk, partNum string
	
	if strings.Contains(rootPart, "nvme") {
		// nvme0n1p2 -> disk=/dev/nvme0n1, part=2
		// Simplified parsing logic
		return fmt.Errorf("NVMe drives not yet supported in this version")
	} else {
		// /dev/sda2
		// Very basic parsing, assumes single digit partition for now
		if len(rootPart) < 4 {
			return fmt.Errorf("unsupported partition format: %s", rootPart)
		}
		disk = rootPart[:len(rootPart)-1] // /dev/sda
		partNum = rootPart[len(rootPart)-1:] // 2
	}

	PrintInfo("Target Disk: %s, Partition: %s", disk, partNum)

	// 3. Grow Partition
	PrintInfo("Extending partition...")
	cmd = exec.Command("growpart", disk, partNum)
	if out, err := cmd.CombinedOutput(); err != nil {
		// Exit code 1 means "no change needed" usually, but let's be careful
		if strings.Contains(string(out), "NOCHANGE") {
			PrintSuccess("Partition is already at max size")
		} else {
			return fmt.Errorf("growpart failed: %v\nOutput: %s", err, string(out))
		}
	} else {
		PrintSuccess("Partition extended")
	}

	// 4. Resize Filesystem
	PrintInfo("Resizing filesystem...")
	
	// Check filesystem type
	cmd = exec.Command("findmnt", "/", "-o", "FSTYPE", "-n")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to detect fs type: %w", err)
	}
	fsType := strings.TrimSpace(string(out))
	PrintInfo("Filesystem: %s", fsType)

	if fsType == "ext4" {
		cmd = exec.Command("resize2fs", rootPart)
	} else if fsType == "xfs" {
		cmd = exec.Command("xfs_growfs", "/")
	} else {
		return fmt.Errorf("unsupported filesystem: %s", fsType)
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("resize failed: %v\nOutput: %s", err, string(out))
	}

	PrintSuccess("Filesystem resized successfully!")
	
	// Show new size
	exec.Command("df", "-h", "/").Run()

	return nil
}
