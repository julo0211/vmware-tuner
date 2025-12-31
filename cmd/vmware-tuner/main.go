package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"vmware-tuner/internal/tuner"
)

var (
	version      = "1.1.0-enterprise"
	dryRun       bool
	noGrub       bool
	noSysctl     bool
	noFstab      bool
	noIO         bool
	noNet        bool
	installTools bool
	doDebloat    bool
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "vmware-tuner",
		Short: "VMware VM Performance Tuning Tool (Enterprise Edition)",
		Long: `VMware VM Performance Tuner (Enterprise)

A comprehensive tool to optimize VMware virtual machines for maximum performance.
Optimized for Air-gapped and Enterprise environments.

Features:
  - Connectivty-aware (Offline Mode)
  - Native JSON Parsing for Disk Operations
  - Native Manifest-based Rollback (No scripts)
  - Proxy Support (HTTP_PROXY)
`,
		Version: version,
		RunE:    runTuner,
	}

	var showCmd = &cobra.Command{
		Use:   "show",
		Short: "Show current system configuration",
		Long:  "Display current system settings for all tuning categories",
		RunE:  showConfig,
	}

	var verifyCmd = &cobra.Command{
		Use:   "verify",
		Short: "Verify tuning has been applied",
		Long:  "Check if tuning configurations are present on the system",
		RunE:  verifyConfig,
	}

	// Root command flags
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without making changes")
	rootCmd.Flags().BoolVar(&noGrub, "no-grub", false, "Skip GRUB boot parameter tuning")
	rootCmd.Flags().BoolVar(&noSysctl, "no-sysctl", false, "Skip sysctl parameter tuning")
	rootCmd.Flags().BoolVar(&noFstab, "no-fstab", false, "Skip fstab optimization")
	rootCmd.Flags().BoolVar(&noIO, "no-io", false, "Skip I/O scheduler tuning")
	rootCmd.Flags().BoolVar(&noNet, "no-network", false, "Skip network tuning")
	rootCmd.Flags().BoolVar(&installTools, "install-tools", true, "Install open-vm-tools if missing")
	rootCmd.Flags().BoolVar(&doDebloat, "debloat", false, "Disable unnecessary services (Server Slim)")

	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(verifyCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runTuner(cmd *cobra.Command, args []string) error {
	tuner.Banner()

	// 1. Check Connectivity
	tuner.PrintStep("Connectivity Check")
	hasInternet := tuner.CheckConnectivity()
	if hasInternet {
		tuner.PrintSuccess("Mode: Connecté (Internet accessible)")
	} else {
		tuner.PrintWarning("Mode: Hors-Ligne (Pas d'accès Internet détecté)")
		tuner.PrintInfo("Certaines fonctionnalités nécessitant internet seront désactivées.")
	}
	fmt.Println()

	// Check if running interactively (no flags)
	if !cmd.Flags().Changed("dry-run") &&
		!cmd.Flags().Changed("no-grub") &&
		!cmd.Flags().Changed("no-sysctl") &&
		!cmd.Flags().Changed("no-fstab") &&
		!cmd.Flags().Changed("no-io") &&
		!cmd.Flags().Changed("no-network") &&
		!cmd.Flags().Changed("install-tools") &&
		!cmd.Flags().Changed("debloat") {

		// Initialize distro manager for all interactive commands
		distro, err := tuner.NewDistroManager()
		if err != nil {
			// Fallback if detection fails
			distro = &tuner.DistroManager{Type: tuner.DistroUnknown}
		}

		// Define Menu Options
		type MenuOption struct {
			Label       string
			Action      func() error
			RequireRoot bool
		}

		menu := map[int]MenuOption{
			1: {"Optimize this VM (Tuning)", func() error {
				return fmt.Errorf("EXIT_TO_TUNE") // Special signal to break loop and continue to tuning
			}, true},
			2: {"Restore a backup (Rollback)", runRollbackInteractive, true},
			3: {"Audit System (Score)", func() error { return tuner.NewAuditTuner(distro).RunAudit() }, true},
			4: {"Expand Disk", func() error { return tuner.NewDiskTuner(distro).ExpandRoot() }, true},
			5: {"Fix Time Sync", func() error { return tuner.NewTimeSyncTuner(distro).Run() }, true},
			6: {"Clean System", func() error { return tuner.NewCleanerTuner(distro).Run() }, true},
			7: {"Secure SSH", func() error {
				backup := tuner.NewBackupManager()
				if err := backup.Initialize(); err != nil {
					return err
				}
				return tuner.NewSSHTuner(backup).Run()
			}, true},
			8:  {"Schedule Maintenance", func() error { return tuner.NewCronTuner().Run() }, true},
			9:  {"System Info", func() error { return tuner.NewInfoTuner().Run() }, false},
			10: {"Network Benchmark", func() error { return tuner.NewBenchmarkTuner().Run() }, false},
			11: {"Seal VM for Template (Expert)", func() error { return tuner.NewTemplateTuner().Run() }, true},
			12: {"Check Virtual Hardware", func() error { return tuner.NewHardwareTuner(distro).Run() }, false},
			13: {"Manage Swap", func() error { return tuner.NewSwapTuner().Run() }, true},
			14: {"Scan Logs for Errors", func() error { return tuner.NewLogDoctorTuner(distro).Run() }, true},
			// 15 is dynamic (Docker)
			// 16 Updated for connectivity awareness
			16: {"Safe System Update", func() error {
				return tuner.NewUpdateTuner(distro).Run(hasInternet)
			}, true},
		}

		// Add Docker option if installed
		if _, err := exec.LookPath("docker"); err == nil {
			menu[15] = MenuOption{"Optimize Docker", func() error { return tuner.NewDockerTuner().Run() }, true}
		}

		for {
			tuner.Banner()
			fmt.Println("What do you want to do?")

			// Print menu items in order
			var keys []int
			for k := range menu {
				keys = append(keys, k)
			}
			sort.Ints(keys)

			for _, k := range keys {
				fmt.Printf("  [%d] %s\n", k, menu[k].Label)
			}
			if _, err := exec.LookPath("docker"); err != nil {
				color.Red("  [15] Optimize Docker (Not Installed)")
			}
			fmt.Println("  [0]  Exit")
			fmt.Println()
			fmt.Print("Choice: ")

			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			if input == "0" {
				tuner.PrintInfo("Exiting...")
				return nil
			}

			choice, err := strconv.Atoi(input)
			if err != nil || choice < 0 {
				tuner.PrintError("Invalid choice")
				tuner.Pause()
				continue
			}

			option, exists := menu[choice]
			if !exists {
				tuner.PrintError("Invalid choice")
				tuner.Pause()
				continue
			}

			if option.RequireRoot {
				if err := tuner.CheckRoot(); err != nil {
					tuner.PrintError("%v", err)
					tuner.Pause()
					continue
				}
			}

			err = option.Action()

			// Check for special exit signal
			if err != nil && err.Error() == "EXIT_TO_TUNE" {
				break // Break loop and continue to main tuning logic
			}

			if err != nil {
				tuner.PrintError("%v", err)
			}

			tuner.Pause()

			// Clear screen for next iteration
			fmt.Print("\033[H\033[2J")
		}
	}

	// --- TUNING LOGIC ---

	// Check if running as root
	if !dryRun {
		if err := tuner.CheckRoot(); err != nil {
			tuner.PrintError("%v", err)
			return err
		}
	}

	// Check if running on VMware
	isVMware, err := tuner.IsVMware("")
	if err != nil {
		tuner.PrintWarning("Could not determine if running on VMware: %v", err)
	} else if !isVMware {
		tuner.PrintWarning("This system does not appear to be a VMware VM")
		tuner.PrintWarning("Tuning parameters are optimized for VMware environments")
		fmt.Print("\nContinue anyway? (yes/no): ")
		var response string
		fmt.Scanln(&response)
		if response != "yes" {
			tuner.PrintInfo("Tuning cancelled")
			return nil
		}
	} else {
		tuner.PrintSuccess("Detected VMware virtual machine")
	}

	// Initialize distro manager
	distro, err := tuner.NewDistroManager()
	if err != nil {
		tuner.PrintWarning("Could not detect distribution: %v", err)
		// Continue with default/unknown
		distro = &tuner.DistroManager{Type: tuner.DistroUnknown}
	} else {
		tuner.PrintSuccess("Detected distribution: %s", distro.Name)
	}

	// Check and install dependencies
	if !dryRun && !noNet {
		if err := distro.InstallPackage("ethtool"); err != nil {
			tuner.PrintWarning("Failed to install ethtool: %v", err)
			tuner.PrintWarning("Network tuning might fail")
		}
	}

	// Determine what will be tuned
	var modules []string
	if !noGrub {
		modules = append(modules, "GRUB boot parameters")
	}
	if !noSysctl {
		modules = append(modules, "Sysctl kernel parameters")
	}
	if !noFstab {
		modules = append(modules, "Filesystem mount options")
	}
	if !noIO {
		modules = append(modules, "I/O scheduler configuration")
	}
	if !noNet {
		modules = append(modules, "Network interface optimization")
	}
	if installTools {
		modules = append(modules, "VMware Tools verification/installation")
	}
	if doDebloat {
		modules = append(modules, "Server Slim (disable unused services)")
	}

	if len(modules) == 0 {
		tuner.PrintError("No tuning modules selected")
		return fmt.Errorf("nothing to do")
	}

	tuner.Summary(modules)

	if dryRun {
		tuner.PrintInfo("DRY RUN MODE - No changes will be made")
		fmt.Println()
	} else {
		fmt.Print("Continue with tuning? (yes/no): ")
		var response string
		fmt.Scanln(&response)
		if response != "yes" {
			tuner.PrintInfo("Tuning cancelled")
			return nil
		}
	}

	// Initialize backup manager
	backup := tuner.NewBackupManager()
	if !dryRun {
		if err := backup.Initialize(); err != nil {
			tuner.PrintError("Failed to initialize backup: %v", err)
			return err
		}
		tuner.PrintSuccess("Backup directory created: %s", backup.BackupDir)
	}

	rebootRequired := false

	// Apply GRUB tuning
	if !noGrub {
		grub := tuner.NewGrubTuner(dryRun, distro)
		if err := grub.Apply(backup); err != nil {
			tuner.PrintError("GRUB tuning failed: %v", err)
		} else {
			rebootRequired = true
		}
	}

	// Apply sysctl tuning
	if !noSysctl {
		sysctl := tuner.NewSysctlTuner(dryRun)
		if err := sysctl.Apply(backup); err != nil {
			tuner.PrintError("Sysctl tuning failed: %v", err)
		}
	}

	// Apply fstab tuning
	if !noFstab {
		fstab := tuner.NewFstabTuner(dryRun)
		if err := fstab.Apply(backup); err != nil {
			tuner.PrintError("Fstab tuning failed: %v", err)
		}
	}

	// Apply I/O scheduler tuning
	if !noIO {
		scheduler := tuner.NewSchedulerTuner(dryRun)
		if err := scheduler.Apply(backup); err != nil {
			tuner.PrintError("I/O scheduler tuning failed: %v", err)
		}
	}

	// Apply network tuning
	if !noNet {
		network := tuner.NewNetworkTuner(dryRun)
		if err := network.Apply(backup); err != nil {
			tuner.PrintError("Network tuning failed: %v", err)
		}
	}

	// Apply VM Tools
	if installTools {
		tools := tuner.NewVMToolsTuner(dryRun, distro)
		// Pass connectivity status to Apply
		if err := tools.Apply(hasInternet); err != nil {
			tuner.PrintError("VM Tools tuning failed: %v", err)
		}
	}

	// Apply Debloat (Interactive or Flag)
	debloat := tuner.NewDebloatTuner(dryRun)
	if doDebloat {
		// Flag provided: do it automatically
		if err := debloat.Apply(backup); err != nil {
			tuner.PrintError("Debloat failed: %v", err)
		}
	} else if !dryRun {
		// No flag: ask interactively
		services := debloat.GetBloatServices()
		if len(services) > 0 {
			tuner.PrintStep("Server Slim Mode (Optional)")
			tuner.PrintInfo("Found %d services that are usually unnecessary on servers:", len(services))
			for _, svc := range services {
				fmt.Printf("  - %s: %s\n", svc.Name, svc.Description)
			}
			fmt.Println()
			fmt.Print("Do you want to disable these services? (y/n): ")
			var response string
			fmt.Scanln(&response)
			if response == "y" || response == "yes" {
				if err := debloat.DisableServices(services, backup); err != nil {
					tuner.PrintError("Debloat failed: %v", err)
				}
			} else {
				tuner.PrintInfo("Skipping Server Slim optimization")
			}
		}
	}

	// Create rollback script (REMOVED - using manifest)
	// if !dryRun {
	// 	if err := backup.CreateRollbackScript(); err != nil {
	// 		tuner.PrintWarning("Failed to create rollback script: %v", err)
	// 	}
	// }

	if !dryRun {
		tuner.CompletionMessage(rebootRequired)

		if rebootRequired {
			fmt.Print("Do you want to reboot now? (y/n): ")
			var response string
			fmt.Scanln(&response)
			if response == "y" || response == "yes" {
				tuner.PrintInfo("Rebooting system...")
				exec.Command("reboot").Run()
			} else {
				tuner.PrintInfo("Please remember to reboot later")
			}
		}
	} else {
		fmt.Println()
		tuner.PrintInfo("DRY RUN completed - no changes were made")
		tuner.PrintInfo("Run without --dry-run to apply changes")
	}

	return nil
}

func showConfig(cmd *cobra.Command, args []string) error {
	tuner.Banner()
	tuner.PrintInfo("Current System Configuration")
	fmt.Println()

	// Initialize distro manager for config paths
	distro, _ := tuner.NewDistroManager()

	// Show GRUB config
	grub := tuner.NewGrubTuner(false, distro)
	if err := grub.ShowCurrent(); err != nil {
		tuner.PrintWarning("Could not show GRUB config: %v", err)
	}

	// Show sysctl config
	sysctl := tuner.NewSysctlTuner(false)
	if err := sysctl.ShowCurrent(); err != nil {
		tuner.PrintWarning("Could not show sysctl config: %v", err)
	}

	// Show fstab config
	fstab := tuner.NewFstabTuner(false)
	if err := fstab.ShowCurrent(); err != nil {
		tuner.PrintWarning("Could not show fstab config: %v", err)
	}

	// Show I/O scheduler config
	scheduler := tuner.NewSchedulerTuner(false)
	if err := scheduler.ShowCurrent(); err != nil {
		tuner.PrintWarning("Could not show I/O scheduler config: %v", err)
	}

	// Show network config
	network := tuner.NewNetworkTuner(false)
	if err := network.ShowCurrent(); err != nil {
		tuner.PrintWarning("Could not show network config: %v", err)
	}

	return nil
}

func verifyConfig(cmd *cobra.Command, args []string) error {
	tuner.Banner()
	tuner.PrintStep("Verifying tuning configuration")

	allGood := true

	// Verify sysctl
	sysctl := tuner.NewSysctlTuner(false)
	if err := sysctl.Verify(); err != nil {
		tuner.PrintWarning("Sysctl: %v", err)
		allGood = false
	}

	// Verify I/O scheduler
	scheduler := tuner.NewSchedulerTuner(false)
	if err := scheduler.Verify(); err != nil {
		tuner.PrintWarning("I/O Scheduler: %v", err)
		allGood = false
	}

	// Verify network
	network := tuner.NewNetworkTuner(false)
	if err := network.Verify(); err != nil {
		tuner.PrintWarning("Network: %v", err)
		allGood = false
	}

	fmt.Println()
	if allGood {
		tuner.PrintSuccess("All tuning configurations are present")
	} else {
		tuner.PrintWarning("Some tuning configurations are missing")
		tuner.PrintInfo("Run 'vmware-tuner' to apply tuning")
	}

	return nil
}

func runRollbackInteractive() error {
	tuner.PrintStep("Restore Backup (Native Rollback)")

	backups, err := tuner.ListBackups()
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	if len(backups) == 0 {
		tuner.PrintWarning("No backups found.")
		return nil
	}

	fmt.Println("Available backups:")
	for i, backup := range backups {
		fmt.Printf("  [%d] %s\n", i+1, backup)
	}
	fmt.Println("  [c] Cancel")
	fmt.Println()

	fmt.Print("Select backup to restore: ")
	var selection string
	fmt.Scanln(&selection)

	if selection == "c" || selection == "C" {
		tuner.PrintInfo("Rollback cancelled")
		return nil
	}

	var index int
	_, err = fmt.Sscanf(selection, "%d", &index)
	if err != nil || index < 1 || index > len(backups) {
		tuner.PrintError("Invalid selection")
		return nil
	}

	targetBackup := backups[index-1]
	backupDir := filepath.Join("/root", ".vmware-tuner-backups", targetBackup)

	// Create a backup manager instance pointing to this directory
	bm := &tuner.BackupManager{
		BackupDir: backupDir,
		Timestamp: targetBackup,
	}

	// Check if manifest exists
	if !tuner.FileExists(filepath.Join(backupDir, "manifest.json")) {
		// Fallback to legacy script if manifest is missing (backward compatibility)
		scriptPath := filepath.Join(backupDir, "rollback.sh")
		if tuner.FileExists(scriptPath) {
			tuner.PrintWarning("Manifest missing, falling back to legacy rollback.sh")
			tuner.PrintInfo("Executing rollback script from %s...", targetBackup)
			cmd := exec.Command("/bin/bash", scriptPath)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin
			return cmd.Run()
		}
		return fmt.Errorf("no manifest or rollback script found in %s", backupDir)
	}

	return bm.RestoreFromManifest()
}
