package tuner

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// FstabTuner handles /etc/fstab optimization
type FstabTuner struct {
	FstabPath string
	DryRun    bool
}

// NewFstabTuner creates a new fstab tuner
func NewFstabTuner(dryRun bool) *FstabTuner {
	return &FstabTuner{
		FstabPath: "/etc/fstab",
		DryRun:    dryRun,
	}
}

// FstabEntry represents a line in /etc/fstab
type FstabEntry struct {
	Device     string
	MountPoint string
	FSType     string
	Options    []string
	Dump       string
	Pass       string
	Comment    string
	IsComment  bool
}

// ParseFstab parses /etc/fstab and returns entries
func (ft *FstabTuner) ParseFstab() ([]FstabEntry, error) {
	file, err := os.Open(ft.FstabPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", ft.FstabPath, err)
	}
	defer file.Close()

	var entries []FstabEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Handle comments and empty lines
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			entries = append(entries, FstabEntry{
				Comment:   line,
				IsComment: true,
			})
			continue
		}

		// Parse fstab entry
		fields := regexp.MustCompile(`\s+`).Split(trimmed, -1)
		if len(fields) < 4 {
			// Malformed line, keep as comment
			entries = append(entries, FstabEntry{
				Comment:   line,
				IsComment: true,
			})
			continue
		}

		entry := FstabEntry{
			Device:     fields[0],
			MountPoint: fields[1],
			FSType:     fields[2],
			Options:    strings.Split(fields[3], ","),
			IsComment:  false,
		}

		if len(fields) > 4 {
			entry.Dump = fields[4]
		} else {
			entry.Dump = "0"
		}

		if len(fields) > 5 {
			entry.Pass = fields[5]
		} else {
			entry.Pass = "0"
		}

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading fstab: %w", err)
	}

	return entries, nil
}

// OptimizeEntry optimizes mount options for a given entry
func (ft *FstabTuner) OptimizeEntry(entry *FstabEntry) bool {
	// Only optimize ext4 filesystems
	if entry.FSType != "ext4" {
		return false
	}

	// Skip swap and special filesystems
	if entry.MountPoint == "none" || entry.FSType == "swap" {
		return false
	}

	modified := false
	options := make(map[string]bool)

	// Parse existing options
	for _, opt := range entry.Options {
		options[opt] = true
	}

	// Remove discard if present (VMware doesn't support it)
	if options["discard"] {
		delete(options, "discard")
		modified = true
	}

	// Add performance options if not present
	performanceOpts := []string{"noatime", "nodiratime"}
	for _, opt := range performanceOpts {
		if !options[opt] {
			options[opt] = true
			modified = true
		}
	}

	// Add commit=60 if not present
	hasCommit := false
	for opt := range options {
		if strings.HasPrefix(opt, "commit=") {
			hasCommit = true
			break
		}
	}
	if !hasCommit {
		options["commit=60"] = true
		modified = true
	}

	// Rebuild options slice
	if modified {
		newOptions := []string{}
		for opt := range options {
			newOptions = append(newOptions, opt)
		}
		entry.Options = newOptions
	}

	return modified
}

// Apply applies fstab optimizations
func (ft *FstabTuner) Apply(backup *BackupManager) error {
	PrintStep("Optimizing /etc/fstab")

	// Parse current fstab
	entries, err := ft.ParseFstab()
	if err != nil {
		return err
	}

	// Optimize entries
	modified := false
	for i := range entries {
		if !entries[i].IsComment {
			if ft.OptimizeEntry(&entries[i]) {
				modified = true
				PrintInfo("Optimizing: %s mounted at %s",
					entries[i].Device, entries[i].MountPoint)
			}
		}
	}

	if !modified {
		PrintSuccess("No fstab optimizations needed")
		return nil
	}

	// Generate new fstab content
	newContent := ft.GenerateFstab(entries)

	if ft.DryRun {
		PrintInfo("Would update: %s", ft.FstabPath)
		PrintInfo("New content preview:")
		fmt.Println(newContent)
		return nil
	}

	// Backup existing fstab
	if err := backup.BackupFile(ft.FstabPath); err != nil {
		return fmt.Errorf("failed to backup fstab: %w", err)
	}

	// Write new fstab
	if err := os.WriteFile(ft.FstabPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write fstab: %w", err)
	}

	PrintSuccess("Updated %s", ft.FstabPath)

	// Remount filesystems with new options
	PrintInfo("Remounting filesystems...")
	for _, entry := range entries {
		if !entry.IsComment && entry.FSType == "ext4" && entry.MountPoint != "none" {
			if err := ft.RemountFilesystem(entry.MountPoint); err != nil {
				PrintWarning("Failed to remount %s: %v", entry.MountPoint, err)
				PrintWarning("A reboot may be required for changes to take effect")
			} else {
				PrintSuccess("Remounted %s", entry.MountPoint)
			}
		}
	}

	return nil
}

// GenerateFstab generates fstab content from entries
func (ft *FstabTuner) GenerateFstab(entries []FstabEntry) string {
	var lines []string

	for _, entry := range entries {
		if entry.IsComment {
			lines = append(lines, entry.Comment)
			continue
		}

		// Format the entry
		optionsStr := strings.Join(entry.Options, ",")
		line := fmt.Sprintf("%-45s %-15s %-7s %-30s %s %s",
			entry.Device,
			entry.MountPoint,
			entry.FSType,
			optionsStr,
			entry.Dump,
			entry.Pass)

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n") + "\n"
}

// RemountFilesystem remounts a filesystem with new options
func (ft *FstabTuner) RemountFilesystem(mountPoint string) error {
	cmd := exec.Command("mount", "-o", "remount", mountPoint)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %w", string(output), err)
	}
	return nil
}

// ShowCurrent displays current fstab configuration
func (ft *FstabTuner) ShowCurrent() error {
	PrintStep("Current /etc/fstab entries")

	entries, err := ft.ParseFstab()
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsComment {
			continue
		}

		fmt.Printf("\n  Mount: %s\n", entry.MountPoint)
		fmt.Printf("  Device: %s\n", entry.Device)
		fmt.Printf("  Type: %s\n", entry.FSType)
		fmt.Printf("  Options: %s\n", strings.Join(entry.Options, ","))
	}

	return nil
}
