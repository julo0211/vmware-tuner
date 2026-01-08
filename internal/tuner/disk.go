package tuner

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
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

// BlockDevice represents a block device from lsblk JSON
type BlockDevice struct {
	Name       string        `json:"name"`
	Type       string        `json:"type"`
	Mountpoint string        `json:"mountpoint"`
	Children   []BlockDevice `json:"children,omitempty"`
}

// LsblkOutput represents the root JSON output from lsblk
type LsblkOutput struct {
	BlockDevices []BlockDevice `json:"blockdevices"`
}

// ExpandRoot expands the root partition and filesystem
func (dt *DiskTuner) ExpandRoot(hasInternet bool) error {
	PrintStep("Disk Expansion Assistant")

	PrintWarning("⚠️  ATTENTION : Les opérations sur disque comportent un risque.")
	PrintWarning("Assurez-vous d'avoir un snapshot ou une sauvegarde avant de continuer.")
	fmt.Println()

	if !AskUser("Voulez-vous continuer ?") {
		PrintInfo("Opération annulée")
		return nil
	}

	// 1. Check/Install dependencies (growpart)
	if _, err := exec.LookPath("growpart"); err != nil {
		PrintWarning("Outil 'growpart' manquant.")

		if !hasInternet {
			return fmt.Errorf("impossible d'installer 'growpart' en mode Hors-Ligne. Veuillez l'installer manuellement (cloud-guest-utils)")
		}

		PrintInfo("Tentative d'installation...")
		if err := dt.Distro.InstallPackage("cloud-guest-utils"); err != nil {
			// Fallback for RHEL-based systems
			if err := dt.Distro.InstallPackage("cloud-utils-growpart"); err != nil {
				return fmt.Errorf("échec de l'installation de growpart: %v", err)
			}
		}
	}

	// 2. Identify root device using lsblk JSON
	PrintInfo("Analyse de la structure disque (JSON)...")

	cmd := exec.Command("lsblk", "-J", "-o", "NAME,TYPE,MOUNTPOINT")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("lsblk failed: %w", err)
	}

	var data LsblkOutput
	if err := json.Unmarshal(output, &data); err != nil {
		return fmt.Errorf("failed to parse lsblk json: %w", err)
	}

	diskName, partNum, err := dt.findRootInTree(data.BlockDevices)
	if err != nil {
		return err
	}

	PrintInfo("Cible détectée -> Disque: /dev/%s, Partition N°: %s", diskName, partNum)

	// 3. Grow Partition
	PrintInfo("Extension de la partition...")
	// growpart /dev/sda 1
	cmd = exec.Command("growpart", "/dev/"+diskName, partNum)
	if out, err := cmd.CombinedOutput(); err != nil {
		if strings.Contains(string(out), "NOCHANGE") {
			PrintSuccess("La partition est déjà à la taille maximale")
		} else {
			return fmt.Errorf("growpart failed: %v\nOutput: %s", err, string(out))
		}
	} else {
		PrintSuccess("Partition étendue avec succès")
	}

	// 4. Resize Filesystem
	PrintInfo("Redimensionnement du système de fichiers...")

	// Detect FS Type
	cmd = exec.Command("findmnt", "/", "-o", "FSTYPE", "-n")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to detect fs type: %w", err)
	}
	fsType := strings.TrimSpace(string(out))
	PrintInfo("Type de FS: %s", fsType)

	// Construct partition path for resize command
	// Logic: /dev/sda + 1 => /dev/sda1, but /dev/nvme0n1 + 1 => /dev/nvme0n1p1
	partPath := "/dev/" + diskName
	if (strings.Contains(diskName, "nvme") || strings.Contains(diskName, "loop") || strings.Contains(diskName, "mmcblk")) && !strings.HasSuffix(diskName, "p") {
		partPath += "p" + partNum
	} else {
		partPath += partNum
	}

	if fsType == "ext4" {
		cmd = exec.Command("resize2fs", partPath)
	} else if fsType == "xfs" {
		cmd = exec.Command("xfs_growfs", "/")
	} else {
		return fmt.Errorf("système de fichiers non supporté pour l'auto-resize: %s", fsType)
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("resize failed: %v\nOutput: %s", err, string(out))
	}

	PrintSuccess("Système de fichiers étendu avec succès !")

	// Show new size
	exec.Command("df", "-h", "/").Run()

	return nil
}

// findRootInTree traverses the BlockDevice tree to find the disk containing /
func (dt *DiskTuner) findRootInTree(devices []BlockDevice) (string, string, error) {
	for _, dev := range devices {
		// Case 1: The device itself is mounted as root (rare, unpartitioned disk)
		if dev.Mountpoint == "/" {
			return dev.Name, "", fmt.Errorf("root is on a raw disk without partitions, dangerous to resize automatically")
		}

		// Case 2: Check children (partitions)
		if len(dev.Children) > 0 {
			for _, child := range dev.Children {
				if child.Mountpoint == "/" {
					// Found it! Parent is 'dev', Child is 'child'
					// We need to extract the partition number from the child name relative to parent
					// e.g., Parent: sda, Child: sda1 -> PartNum: 1
					partNum := dt.extractPartitionNumber(dev.Name, child.Name)
					return dev.Name, partNum, nil
				}

				// Handle LVM (Child might be a Volume Group container)
				if len(child.Children) > 0 {
					// Recursive check not fully implemented for deep LVM layers for safety
					// But we can check if LVM logical volume is root
					for _, lv := range child.Children {
						if lv.Mountpoint == "/" {
							return "", "", fmt.Errorf("LVM detected. Automated LVM resizing is disabled for safety")
						}
					}
				}
			}
		}
	}
	return "", "", fmt.Errorf("root partition not found in disk tree")
}

func (dt *DiskTuner) extractPartitionNumber(disk, partition string) string {
	// Simple heuristic: remove the disk name from the partition name
	// sda1 - sda = 1
	// nvme0n1p1 - nvme0n1 = p1 -> clean to 1

	suffix := strings.TrimPrefix(partition, disk)

	// Remove 'p' separator if present (nvme0n1p1 -> 1)
	if strings.HasPrefix(suffix, "p") {
		suffix = strings.TrimPrefix(suffix, "p")
	}

	// Extract digits
	re := regexp.MustCompile(`\d+`)
	match := re.FindString(suffix)
	if match != "" {
		return match
	}

	return "1" // Fallback
}
