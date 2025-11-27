package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// BackupManager handles configuration file backups
type BackupManager struct {
	BackupDir string
	Timestamp string
}

// NewBackupManager creates a new backup manager
func NewBackupManager() *BackupManager {
	timestamp := time.Now().Format("20060102-150405")
	backupDir := filepath.Join("/root", ".vmware-tuner-backups", timestamp)

	return &BackupManager{
		BackupDir: backupDir,
		Timestamp: timestamp,
	}
}

// Initialize creates the backup directory
func (bm *BackupManager) Initialize() error {
	if err := os.MkdirAll(bm.BackupDir, 0700); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create a README in the backup directory
	readme := filepath.Join(bm.BackupDir, "README.txt")
	content := fmt.Sprintf(`VMware Tuner Backup
Created: %s

This directory contains backups of system configuration files
before they were modified by vmware-tuner.

To restore a file:
  sudo cp <filename> /path/to/original/location

To restore all files, run:
  sudo vmware-tuner --rollback %s
`, time.Now().Format(time.RFC3339), bm.Timestamp)

	if err := os.WriteFile(readme, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create README: %w", err)
	}

	return nil
}

// BackupFile creates a backup of the specified file
func (bm *BackupManager) BackupFile(filePath string) error {
	// Check if source file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// File doesn't exist, nothing to backup
		return nil
	}

	// Open source file
	source, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", filePath, err)
	}
	defer source.Close()

	// Create backup filename (preserve directory structure)
	backupPath := filepath.Join(bm.BackupDir, filepath.Base(filePath))

	// Create backup file
	backup, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file %s: %w", backupPath, err)
	}
	defer backup.Close()

	// Copy contents
	if _, err := io.Copy(backup, source); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	// Preserve permissions
	sourceInfo, err := os.Stat(filePath)
	if err == nil {
		if err := os.Chmod(backupPath, sourceInfo.Mode()); err != nil {
			return fmt.Errorf("failed to preserve permissions: %w", err)
		}
	}

	return nil
}

// CreateRollbackScript creates a script to restore all backed up files
func (bm *BackupManager) CreateRollbackScript() error {
	scriptPath := filepath.Join(bm.BackupDir, "rollback.sh")

	script := `#!/bin/bash
# VMware Tuner Rollback Script
# This script restores all configuration files to their pre-tuning state

set -e

echo "VMware Tuner - Rollback Script"
echo "==============================="
echo ""
echo "This will restore the following files:"
echo ""

# Show what will be restored
if [ -f "fstab" ]; then
    echo "  - /etc/fstab"
fi
if [ -f "grub" ]; then
    echo "  - /etc/default/grub"
fi
if [ -f "99-vmware-performance.conf" ]; then
    echo "  - /etc/sysctl.d/99-vmware-performance.conf"
fi
if [ -f "60-scheduler.rules" ]; then
    echo "  - /etc/udev/rules.d/60-scheduler.rules"
fi

echo ""
read -p "Continue with rollback? (yes/no): " confirm

if [ "$confirm" != "yes" ]; then
    echo "Rollback cancelled."
    exit 0
fi

echo ""
echo "Restoring files..."

if [ -f "fstab" ]; then
    cp -v fstab /etc/fstab
    echo "✓ Restored /etc/fstab"
fi

if [ -f "grub" ]; then
    cp -v grub /etc/default/grub
    update-grub
    echo "✓ Restored /etc/default/grub (grub updated)"
fi

if [ -f "99-vmware-performance.conf" ]; then
    rm -f /etc/sysctl.d/99-vmware-performance.conf
    echo "✓ Removed /etc/sysctl.d/99-vmware-performance.conf"
    sysctl --system
    echo "✓ Reloaded sysctl configuration"
fi

if [ -f "60-scheduler.rules" ]; then
    rm -f /etc/udev/rules.d/60-scheduler.rules
    udevadm control --reload-rules
    echo "✓ Removed /etc/udev/rules.d/60-scheduler.rules"
    echo "✓ Removed /etc/udev/rules.d/60-scheduler.rules"
fi

if [ -f "services.txt" ]; then
    echo ""
    echo "Re-enabling disabled services..."
    while IFS= read -r service; do
        if [ ! -z "$service" ]; then
            systemctl enable --now "$service"
            echo "✓ Re-enabled $service"
        fi
    done < "services.txt"
fi

echo ""
echo "==============================="
echo "Rollback completed successfully!"
echo ""
echo "Note: Some changes require a reboot to take full effect."
echo "Run 'sudo reboot' when ready."
`

	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		return fmt.Errorf("failed to create rollback script: %w", err)
	}

	return nil
}

// GetBackupPath returns the full path to a backed up file
func (bm *BackupManager) GetBackupPath(filename string) string {
	return filepath.Join(bm.BackupDir, filename)
}

// ListBackups lists all available backup timestamps
func ListBackups() ([]string, error) {
	backupRoot := "/root/.vmware-tuner-backups"

	if _, err := os.Stat(backupRoot); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(backupRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backups []string
	for _, entry := range entries {
		if entry.IsDir() {
			backups = append(backups, entry.Name())
		}
	}

	return backups, nil
}

// BackupServices saves a list of disabled services
func (bm *BackupManager) BackupServices(services []string) error {
	if len(services) == 0 {
		return nil
	}

	filePath := filepath.Join(bm.BackupDir, "services.txt")
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create services backup: %w", err)
	}
	defer file.Close()

	for _, svc := range services {
		if _, err := file.WriteString(svc + "\n"); err != nil {
			return fmt.Errorf("failed to write service to backup: %w", err)
		}
	}

	return nil
}
