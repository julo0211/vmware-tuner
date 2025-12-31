package tuner

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// BackupManager handles configuration file backups
type BackupManager struct {
	BackupDir string
	Timestamp string
}

// ManifestEntry represents a single backed up file
type ManifestEntry struct {
	OriginalPath string      `json:"original_path"`
	BackupPath   string      `json:"backup_path"`
	Mode         os.FileMode `json:"mode"`
}

// Manifest represents the backup manifest
type Manifest struct {
	Timestamp string          `json:"timestamp"`
	Entries   []ManifestEntry `json:"entries"`
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
	return nil
}

// BackupFile creates a backup of the specified file
func (bm *BackupManager) BackupFile(filePath string) error {
	// Check if source file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}

	source, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", filePath, err)
	}
	defer source.Close()

	// Create backup filename
	backupFileName := filepath.Base(filePath)
	backupPath := filepath.Join(bm.BackupDir, backupFileName)

	backup, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer backup.Close()

	if _, err := io.Copy(backup, source); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Preserve permissions
	sourceInfo, err := os.Stat(filePath)
	if err == nil {
		os.Chmod(backupPath, sourceInfo.Mode())

		// Update Manifest
		if err := bm.AddEntry(filePath, backupFileName, sourceInfo); err != nil {
			PrintWarning("Failed to update manifest: %v", err)
		}
	}

	return nil
}

// AddEntry adds a file entry to the manifest.json
func (bm *BackupManager) AddEntry(original, backupName string, info os.FileInfo) error {
	manifestPath := filepath.Join(bm.BackupDir, "manifest.json")

	var manifest Manifest

	// Read existing manifest or create new
	data, err := os.ReadFile(manifestPath)
	if err == nil {
		json.Unmarshal(data, &manifest)
	} else {
		manifest.Timestamp = bm.Timestamp
		manifest.Entries = []ManifestEntry{}
	}

	entry := ManifestEntry{
		OriginalPath: original,
		BackupPath:   backupName,
		Mode:         info.Mode(),
	}

	manifest.Entries = append(manifest.Entries, entry)

	newData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	return os.WriteFile(manifestPath, newData, 0644)
}

// RestoreFromManifest restores files based on the manifest.json
func (bm *BackupManager) RestoreFromManifest() error {
	manifestPath := filepath.Join(bm.BackupDir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("manifest not found: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	PrintInfo("Restauration du backup du %s...", manifest.Timestamp)

	for _, entry := range manifest.Entries {
		srcPath := filepath.Join(bm.BackupDir, entry.BackupPath)
		destPath := entry.OriginalPath

		PrintInfo("Restauration %s -> %s", entry.BackupPath, destPath)

		src, err := os.Open(srcPath)
		if err != nil {
			PrintError("Impossible d'ouvrir le fichier backup %s: %v", srcPath, err)
			continue
		}

		// Open dest with truncation
		dest, err := os.OpenFile(destPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, entry.Mode)
		if err != nil {
			src.Close()
			PrintError("Impossible d'écrire sur la destination %s: %v", destPath, err)
			continue
		}

		if _, err := io.Copy(dest, src); err != nil {
			PrintError("Erreur de copie vers %s: %v", destPath, err)
		}

		dest.Chmod(entry.Mode)
		src.Close()
		dest.Close()
	}

	// Trigger system reloads
	exec.Command("systemctl", "daemon-reload").Run()
	if _, err := os.Stat("/etc/default/grub"); err == nil {
		if _, err := exec.LookPath("update-grub"); err == nil {
			exec.Command("update-grub").Run()
		} else {
			// RHEL fallback logic if needed
			exec.Command("grub2-mkconfig", "-o", "/boot/grub2/grub.cfg").Run()
		}
	}
	exec.Command("sysctl", "--system").Run()

	PrintSuccess("Restauration terminée.")
	return nil
}

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
		return nil, err
	}

	var backups []string
	for _, entry := range entries {
		if entry.IsDir() {
			backups = append(backups, entry.Name())
		}
	}
	return backups, nil
}

func (bm *BackupManager) BackupServices(services []string) error {
	// Not used in manifest logic directly but kept for compatibility
	return nil
}
