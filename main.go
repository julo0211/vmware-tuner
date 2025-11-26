package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	version = "1.0.0"
	dryRun  bool
	noGrub  bool
	noSysctl bool
	noFstab bool
	noIO    bool
	noNet   bool
	installTools bool
	doDebloat    bool
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "vmware-tuner",
		Short: "VMware VM Performance Tuning Tool",
		Long: `VMware VM Performance Tuner

A comprehensive tool to optimize VMware virtual machines for maximum performance.
This tool applies industry-standard best practices including:
  - Kernel boot parameter optimization
  - Sysctl tuning for memory and network
  - Filesystem mount options optimization
  - I/O scheduler configuration
  - Network interface optimization

All changes are backed up and can be rolled back.`,
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
	Banner()

	// Check if running interactively (no flags)
	if !cmd.Flags().Changed("dry-run") &&
		!cmd.Flags().Changed("no-grub") &&
		!cmd.Flags().Changed("no-sysctl") &&
		!cmd.Flags().Changed("no-fstab") &&
		!cmd.Flags().Changed("no-io") &&
		!cmd.Flags().Changed("no-network") &&
		!cmd.Flags().Changed("install-tools") &&
		!cmd.Flags().Changed("debloat") {

		choice := showMainMenu()
		
		// Initialize distro manager for all interactive commands
		distro, err := NewDistroManager()
		if err != nil {
			// Fallback if detection fails, though unlikely to work well for disk/audit
			distro = &DistroManager{Type: DistroUnknown}
		}

		if choice == 0 {
			PrintInfo("Exiting...")
			return nil
		}

		if choice == 2 {
			if err := CheckRoot(); err != nil {
				PrintError("%v", err)
				return err
			}
			return runRollbackInteractive()
		}
		if choice == 3 {
			if err := CheckRoot(); err != nil {
				PrintError("%v", err)
				return err
			}
			audit := NewAuditTuner(distro)
			return audit.RunAudit()
		}
		if choice == 4 {
			if err := CheckRoot(); err != nil {
				PrintError("%v", err)
				return err
			}
			disk := NewDiskTuner(distro)
			return disk.ExpandRoot()
		}
		if choice == 5 {
			if err := CheckRoot(); err != nil {
				PrintError("%v", err)
				return err
			}
			timeSync := NewTimeSyncTuner(distro)
			return timeSync.Run()
		}
		if choice == 6 {
			if err := CheckRoot(); err != nil {
				PrintError("%v", err)
				return err
			}
			cleaner := NewCleanerTuner(distro)
			return cleaner.Run()
		}
		if choice == 7 {
			if err := CheckRoot(); err != nil {
				PrintError("%v", err)
				return err
			}
			// Need backup manager for SSH
			backup := NewBackupManager()
			if err := backup.Initialize(); err != nil {
				return err
			}
			ssh := NewSSHTuner(backup)
			return ssh.Run()
		}
		if choice == 8 {
			if err := CheckRoot(); err != nil {
				PrintError("%v", err)
				return err
			}
			cron := NewCronTuner()
			return cron.Run()
		}
		if choice == 9 {
			// No root needed for info strictly speaking, but good for consistency
			info := NewInfoTuner()
			return info.Run()
		}
		if choice == 10 {
			// No root needed for benchmark
			bench := NewBenchmarkTuner()
			return bench.Run()
		}
		if choice == 11 {
			if err := CheckRoot(); err != nil {
				PrintError("%v", err)
				return err
			}
			template := NewTemplateTuner()
			return template.Run()
		}
		if choice == 12 {
			// No root needed strictly, but lspci might need it for full info
			hardware := NewHardwareTuner(distro)
			return hardware.Run()
		}
		if choice == 13 {
			if err := CheckRoot(); err != nil {
				PrintError("%v", err)
				return err
			}
			swap := NewSwapTuner()
			return swap.Run()
		}
		if choice == 14 {
			if err := CheckRoot(); err != nil {
				PrintError("%v", err)
				return err
			}
			logDoctor := NewLogDoctorTuner(distro)
			return logDoctor.Run()
		}
		if choice == 15 {
			if err := CheckRoot(); err != nil {
				PrintError("%v", err)
				return err
			}
			docker := NewDockerTuner()
			return docker.Run()
		}
		if choice == 16 {
			if err := CheckRoot(); err != nil {
				PrintError("%v", err)
				return err
			}
			update := NewUpdateTuner(distro)
			return update.Run()
		}
	}

	// Check if running as root
	if !dryRun {
		if err := CheckRoot(); err != nil {
			PrintError("%v", err)
			return err
		}
	}

	// Check if running on VMware
	isVMware, err := IsVMware()
	if err != nil {
		PrintWarning("Could not determine if running on VMware: %v", err)
	} else if !isVMware {
		PrintWarning("This system does not appear to be a VMware VM")
		PrintWarning("Tuning parameters are optimized for VMware environments")
		fmt.Print("\nContinue anyway? (yes/no): ")
		var response string
		fmt.Scanln(&response)
		if response != "yes" {
			PrintInfo("Tuning cancelled")
			return nil
		}
	} else {
		PrintSuccess("Detected VMware virtual machine")
	}

	// Initialize distro manager
	distro, err := NewDistroManager()
	if err != nil {
		PrintWarning("Could not detect distribution: %v", err)
		// Continue with default/unknown
		distro = &DistroManager{Type: DistroUnknown}
	} else {
		PrintSuccess("Detected distribution: %s", distro.Name)
	}

	// Check and install dependencies
	if !dryRun && !noNet {
		if err := distro.InstallPackage("ethtool"); err != nil {
			PrintWarning("Failed to install ethtool: %v", err)
			PrintWarning("Network tuning might fail")
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
		PrintError("No tuning modules selected")
		return fmt.Errorf("nothing to do")
	}

	Summary(modules)

	if dryRun {
		PrintInfo("DRY RUN MODE - No changes will be made")
		fmt.Println()
	} else {
		fmt.Print("Continue with tuning? (yes/no): ")
		var response string
		fmt.Scanln(&response)
		if response != "yes" {
			PrintInfo("Tuning cancelled")
			return nil
		}
	}

	// Initialize backup manager
	backup := NewBackupManager()
	if !dryRun {
		if err := backup.Initialize(); err != nil {
			PrintError("Failed to initialize backup: %v", err)
			return err
		}
		PrintSuccess("Backup directory created: %s", backup.BackupDir)
	}

	rebootRequired := false

	// Apply GRUB tuning
	if !noGrub {
		grub := NewGrubTuner(dryRun, distro)
		if err := grub.Apply(backup); err != nil {
			PrintError("GRUB tuning failed: %v", err)
		} else {
			rebootRequired = true
		}
	}

	// Apply sysctl tuning
	if !noSysctl {
		sysctl := NewSysctlTuner(dryRun)
		if err := sysctl.Apply(backup); err != nil {
			PrintError("Sysctl tuning failed: %v", err)
		}
	}

	// Apply fstab tuning
	if !noFstab {
		fstab := NewFstabTuner(dryRun)
		if err := fstab.Apply(backup); err != nil {
			PrintError("Fstab tuning failed: %v", err)
		}
	}

	// Apply I/O scheduler tuning
	if !noIO {
		scheduler := NewSchedulerTuner(dryRun)
		if err := scheduler.Apply(backup); err != nil {
			PrintError("I/O scheduler tuning failed: %v", err)
		}
	}

	// Apply network tuning
	if !noNet {
		network := NewNetworkTuner(dryRun)
		if err := network.Apply(backup); err != nil {
			PrintError("Network tuning failed: %v", err)
		}
	}

	// Apply VM Tools
	if installTools {
		tools := NewVMToolsTuner(dryRun, distro)
		if err := tools.Apply(); err != nil {
			PrintError("VM Tools tuning failed: %v", err)
		}
	}

	// Apply Debloat (Interactive or Flag)
	debloat := NewDebloatTuner(dryRun)
	if doDebloat {
		// Flag provided: do it automatically
		if err := debloat.Apply(backup); err != nil {
			PrintError("Debloat failed: %v", err)
		}
	} else if !dryRun {
		// No flag: ask interactively
		services := debloat.GetBloatServices()
		if len(services) > 0 {
			PrintStep("Server Slim Mode (Optional)")
			PrintInfo("Found %d services that are usually unnecessary on servers:", len(services))
			for _, svc := range services {
				fmt.Printf("  - %s: %s\n", svc.Name, svc.Description)
			}
			fmt.Println()
			fmt.Print("Do you want to disable these services? (y/n): ")
			var response string
			fmt.Scanln(&response)
			if response == "y" || response == "yes" {
				if err := debloat.DisableServices(services, backup); err != nil {
					PrintError("Debloat failed: %v", err)
				}
			} else {
				PrintInfo("Skipping Server Slim optimization")
			}
		}
	}

	// Create rollback script
	if !dryRun {
		if err := backup.CreateRollbackScript(); err != nil {
			PrintWarning("Failed to create rollback script: %v", err)
		}
	}

	if !dryRun {
		CompletionMessage(rebootRequired)
		
		if rebootRequired {
			fmt.Print("Do you want to reboot now? (y/n): ")
			var response string
			fmt.Scanln(&response)
			if response == "y" || response == "yes" {
				PrintInfo("Rebooting system...")
				exec.Command("reboot").Run()
			} else {
				PrintInfo("Please remember to reboot later")
			}
		}
	} else {
		fmt.Println()
		PrintInfo("DRY RUN completed - no changes were made")
		PrintInfo("Run without --dry-run to apply changes")
	}

	return nil
}

func showConfig(cmd *cobra.Command, args []string) error {
	Banner()
	PrintInfo("Current System Configuration")
	fmt.Println()

	// Initialize distro manager for config paths
	distro, _ := NewDistroManager()

	// Show GRUB config
	grub := NewGrubTuner(false, distro)
	if err := grub.ShowCurrent(); err != nil {
		PrintWarning("Could not show GRUB config: %v", err)
	}

	// Show sysctl config
	sysctl := NewSysctlTuner(false)
	if err := sysctl.ShowCurrent(); err != nil {
		PrintWarning("Could not show sysctl config: %v", err)
	}

	// Show fstab config
	fstab := NewFstabTuner(false)
	if err := fstab.ShowCurrent(); err != nil {
		PrintWarning("Could not show fstab config: %v", err)
	}

	// Show I/O scheduler config
	scheduler := NewSchedulerTuner(false)
	if err := scheduler.ShowCurrent(); err != nil {
		PrintWarning("Could not show I/O scheduler config: %v", err)
	}

	// Show network config
	network := NewNetworkTuner(false)
	if err := network.ShowCurrent(); err != nil {
		PrintWarning("Could not show network config: %v", err)
	}

	return nil
}

func verifyConfig(cmd *cobra.Command, args []string) error {
	Banner()
	PrintStep("Verifying tuning configuration")

	allGood := true

	// Verify sysctl
	sysctl := NewSysctlTuner(false)
	if err := sysctl.Verify(); err != nil {
		PrintWarning("Sysctl: %v", err)
		allGood = false
	}

	// Verify I/O scheduler
	scheduler := NewSchedulerTuner(false)
	if err := scheduler.Verify(); err != nil {
		PrintWarning("I/O Scheduler: %v", err)
		allGood = false
	}

	// Verify network
	network := NewNetworkTuner(false)
	if err := network.Verify(); err != nil {
		PrintWarning("Network: %v", err)
		allGood = false
	}

	fmt.Println()
	if allGood {
		PrintSuccess("All tuning configurations are present")
	} else {
		PrintWarning("Some tuning configurations are missing")
		PrintInfo("Run 'vmware-tuner' to apply tuning")
	}

	return nil
}
