package tuner

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// GrubTuner handles GRUB boot parameter optimization
type GrubTuner struct {
	GrubPath string
	DryRun   bool
	Distro   *DistroManager
}

// NewGrubTuner creates a new GRUB tuner
func NewGrubTuner(dryRun bool, distro *DistroManager) *GrubTuner {
	path := "/etc/default/grub"
	if distro != nil {
		path = distro.GetGrubConfigPath()
	}
	
	return &GrubTuner{
		GrubPath: path,
		DryRun:   dryRun,
		Distro:   distro,
	}
}

// VMwareBootParams returns optimal boot parameters for VMware VMs
func (gt *GrubTuner) VMwareBootParams() []string {
	return []string{
		"elevator=noop",                    // I/O scheduler for VMs
		"transparent_hugepage=madvise",     // Reduce memory fragmentation
		"vsyscall=emulate",                 // VMware compatibility
		"clocksource=tsc",                  // Use TSC for time
		"tsc=reliable",                     // Trust TSC
		"intel_idle.max_cstate=0",          // Disable deep C-states
		"processor.max_cstate=1",           // Keep CPU responsive
		"nmi_watchdog=0",                   // Disable NMI watchdog (save CPU)
		"pcie_aspm=off",                    // Disable PCIe power management
		"nvme_core.default_ps_max_latency_us=0", // Disable NVMe power save
	}
}

// ParseGrubConfig parses GRUB configuration
func (gt *GrubTuner) ParseGrubConfig() (map[string]string, []string, error) {
	file, err := os.Open(gt.GrubPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open %s: %w", gt.GrubPath, err)
	}
	defer file.Close()

	config := make(map[string]string)
	var lines []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)

		// Parse key=value pairs
		if strings.HasPrefix(strings.TrimSpace(line), "#") || !strings.Contains(line, "=") {
			continue
		}

		// Match GRUB_* variables
		re := regexp.MustCompile(`^([A-Z_]+)=(.*)$`)
		matches := re.FindStringSubmatch(strings.TrimSpace(line))
		if len(matches) == 3 {
			key := matches[1]
			value := strings.Trim(matches[2], `"`)
			config[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("error reading grub config: %w", err)
	}

	return config, lines, nil
}

// Apply applies GRUB optimizations
func (gt *GrubTuner) Apply(backup *BackupManager) error {
	PrintStep("Optimizing GRUB boot parameters")

	// Parse current GRUB config
	config, lines, err := gt.ParseGrubConfig()
	if err != nil {
		return err
	}

	// Get current cmdline
	currentCmdline := config["GRUB_CMDLINE_LINUX_DEFAULT"]
	currentParams := gt.parseParams(currentCmdline)

	// Get VMware optimal params
	vmwareParams := gt.VMwareBootParams()

	// Merge parameters
	newParams := gt.mergeParams(currentParams, vmwareParams)
	newCmdline := strings.Join(newParams, " ")

	// Check if modification is needed
	if currentCmdline == newCmdline {
		PrintSuccess("GRUB boot parameters already optimized")
		return nil
	}

	PrintInfo("Current cmdline: %s", currentCmdline)
	PrintInfo("New cmdline: %s", newCmdline)

	if gt.DryRun {
		PrintInfo("Would update: %s", gt.GrubPath)
		return nil
	}

	// Backup existing GRUB config
	if err := backup.BackupFile(gt.GrubPath); err != nil {
		return fmt.Errorf("failed to backup grub config: %w", err)
	}

	// Update GRUB configuration
	newLines := gt.updateGrubLines(lines, newCmdline)
	newContent := strings.Join(newLines, "\n") + "\n"

	if err := os.WriteFile(gt.GrubPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write grub config: %w", err)
	}

	PrintSuccess("Updated %s", gt.GrubPath)

	// Run update-grub
	PrintInfo("Updating GRUB configuration...")
	if err := gt.Distro.UpdateGrub(); err != nil {
		PrintWarning("Failed to update GRUB: %v", err)
		return fmt.Errorf("grub update failed: %w", err)
	}

	PrintSuccess("GRUB configuration updated")
	PrintWarning("REBOOT REQUIRED for boot parameter changes to take effect")

	return nil
}

// parseParams parses a space-separated parameter string
func (gt *GrubTuner) parseParams(cmdline string) []string {
	if cmdline == "" {
		return []string{}
	}

	// Split by whitespace
	params := strings.Fields(cmdline)
	return params
}

// mergeParams merges existing and new parameters
func (gt *GrubTuner) mergeParams(existing, new []string) []string {
	// Create a map to track parameter keys
	paramMap := make(map[string]string)

	// Extract key from param (handle key=value and standalone params)
	getKey := func(param string) string {
		if idx := strings.Index(param, "="); idx != -1 {
			return param[:idx]
		}
		return param
	}

	// Add existing params
	for _, param := range existing {
		key := getKey(param)
		paramMap[key] = param
	}

	// Add/override with new params
	for _, param := range new {
		key := getKey(param)
		paramMap[key] = param
	}

	// Convert back to slice
	var result []string
	for _, param := range paramMap {
		result = append(result, param)
	}

	return result
}

// updateGrubLines updates GRUB_CMDLINE_LINUX_DEFAULT in the config lines
func (gt *GrubTuner) updateGrubLines(lines []string, newCmdline string) []string {
	var newLines []string
	re := regexp.MustCompile(`^GRUB_CMDLINE_LINUX_DEFAULT=`)

	for _, line := range lines {
		if re.MatchString(strings.TrimSpace(line)) {
			newLines = append(newLines, fmt.Sprintf(`GRUB_CMDLINE_LINUX_DEFAULT="%s"`, newCmdline))
		} else {
			newLines = append(newLines, line)
		}
	}

	return newLines
}

// ShowCurrent displays current boot parameters
func (gt *GrubTuner) ShowCurrent() error {
	PrintStep("Current GRUB configuration")

	config, _, err := gt.ParseGrubConfig()
	if err != nil {
		return err
	}

	cmdline := config["GRUB_CMDLINE_LINUX_DEFAULT"]
	params := gt.parseParams(cmdline)

	fmt.Printf("  GRUB_CMDLINE_LINUX_DEFAULT=\"%s\"\n\n", cmdline)
	fmt.Println("  Boot parameters:")
	for _, param := range params {
		fmt.Printf("    - %s\n", param)
	}

	// Also show current running kernel parameters
	PrintStep("Current running kernel parameters")
	data, err := os.ReadFile("/proc/cmdline")
	if err != nil {
		return err
	}

	fmt.Printf("  %s\n", strings.TrimSpace(string(data)))

	return nil
}
