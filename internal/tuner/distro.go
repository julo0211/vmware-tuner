package tuner

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// DistroType represents the Linux distribution family
type DistroType int

const (
	DistroUnknown DistroType = iota
	DistroDebian             // Debian, Ubuntu, Kali, Mint
	DistroRHEL               // RHEL, CentOS, Fedora, AlmaLinux, Rocky
)

// DistroManager handles distribution-specific operations
type DistroManager struct {
	Type DistroType
	Name string
}

// NewDistroManager creates a new distribution manager
func NewDistroManager() (*DistroManager, error) {
	dm := &DistroManager{
		Type: DistroUnknown,
	}

	if err := dm.detect(); err != nil {
		return nil, err
	}

	return dm, nil
}

// detect determines the running Linux distribution
func (dm *DistroManager) detect() error {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return fmt.Errorf("failed to read /etc/os-release: %w", err)
	}

	content := string(data)
	contentLower := strings.ToLower(content)

	if strings.Contains(contentLower, "debian") || strings.Contains(contentLower, "ubuntu") {
		dm.Type = DistroDebian
		dm.Name = "Debian/Ubuntu"
	} else if strings.Contains(contentLower, "rhel") || strings.Contains(contentLower, "centos") || 
		strings.Contains(contentLower, "fedora") || strings.Contains(contentLower, "almalinux") || 
		strings.Contains(contentLower, "rocky") {
		dm.Type = DistroRHEL
		dm.Name = "RHEL/CentOS"
	} else {
		// Fallback: check for package managers
		if _, err := exec.LookPath("apt-get"); err == nil {
			dm.Type = DistroDebian
			dm.Name = "Debian-based"
		} else if _, err := exec.LookPath("yum"); err == nil {
			dm.Type = DistroRHEL
			dm.Name = "RHEL-based"
		} else if _, err := exec.LookPath("dnf"); err == nil {
			dm.Type = DistroRHEL
			dm.Name = "RHEL-based"
		} else {
			return fmt.Errorf("unsupported distribution")
		}
	}

	return nil
}

// InstallPackage installs a package using the system package manager
func (dm *DistroManager) InstallPackage(pkg string) error {
	var cmd *exec.Cmd

	switch dm.Type {
	case DistroDebian:
		// Update apt cache first? Maybe too slow. Just try install.
		// apt-get install -y <pkg>
		cmd = exec.Command("apt-get", "install", "-y", pkg)
	case DistroRHEL:
		// dnf install -y <pkg> (or yum)
		if _, err := exec.LookPath("dnf"); err == nil {
			cmd = exec.Command("dnf", "install", "-y", pkg)
		} else {
			cmd = exec.Command("yum", "install", "-y", pkg)
		}
	default:
		return fmt.Errorf("unknown distribution type")
	}

	PrintInfo("Installing package %s...", pkg)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install %s: %v\nOutput: %s", pkg, err, string(output))
	}

	PrintSuccess("Installed %s", pkg)
	return nil
}

// UpdateGrub updates the GRUB configuration
func (dm *DistroManager) UpdateGrub() error {
	switch dm.Type {
	case DistroDebian:
		cmd := exec.Command("update-grub")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("update-grub failed: %v\nOutput: %s", err, string(output))
		}
		return nil

	case DistroRHEL:
		// Detect correct output path for grub2-mkconfig
		outputPath := "/boot/grub2/grub.cfg"
		
		// Check for UEFI
		if _, err := os.Stat("/sys/firmware/efi"); err == nil {
			// UEFI detected
			// RHEL 7/8/9 location variations
			// Common paths: /boot/efi/EFI/redhat/grub.cfg, /boot/efi/EFI/centos/grub.cfg
			
			// Try to find the correct path
			candidates := []string{
				"/boot/efi/EFI/redhat/grub.cfg",
				"/boot/efi/EFI/centos/grub.cfg",
				"/boot/efi/EFI/almalinux/grub.cfg",
				"/boot/efi/EFI/rocky/grub.cfg",
				"/boot/efi/EFI/fedora/grub.cfg",
			}
			
			found := false
			for _, path := range candidates {
				if _, err := os.Stat(path); err == nil {
					outputPath = path
					found = true
					break
				}
			}
			
			// On newer RHEL (9.3+), /boot/grub2/grub.cfg might be the unified location even for EFI
			// If no specific EFI file found, stick to /boot/grub2/grub.cfg or try to detect if it's a symlink?
			// For now, if not found in EFI partition, default to /boot/grub2/grub.cfg
			if !found {
				PrintWarning("Could not detect specific EFI GRUB path, defaulting to %s", outputPath)
			}
		}

		PrintInfo("Updating GRUB config at %s...", outputPath)
		cmd := exec.Command("grub2-mkconfig", "-o", outputPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("grub2-mkconfig failed: %v\nOutput: %s", err, string(output))
		}
		return nil

	default:
		return fmt.Errorf("unsupported distribution for GRUB update")
	}
}

// GetGrubConfigPath returns the path to the GRUB configuration file
func (dm *DistroManager) GetGrubConfigPath() string {
	// Usually /etc/default/grub for both
	return "/etc/default/grub"
}
