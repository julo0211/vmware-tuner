package main

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
)

var (
	// Color definitions
	colorSuccess = color.New(color.FgGreen, color.Bold)
	colorError   = color.New(color.FgRed, color.Bold)
	colorWarning = color.New(color.FgYellow, color.Bold)
	colorInfo    = color.New(color.FgCyan)
	colorStep    = color.New(color.FgMagenta, color.Bold)
)

// PrintSuccess prints a success message
func PrintSuccess(format string, args ...interface{}) {
	colorSuccess.Print("✓ ")
	fmt.Printf(format+"\n", args...)
}

// PrintError prints an error message
func PrintError(format string, args ...interface{}) {
	colorError.Print("✗ ")
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// PrintWarning prints a warning message
func PrintWarning(format string, args ...interface{}) {
	colorWarning.Print("⚠ ")
	fmt.Printf(format+"\n", args...)
}

// PrintInfo prints an info message
func PrintInfo(format string, args ...interface{}) {
	colorInfo.Print("ℹ ")
	fmt.Printf(format+"\n", args...)
}

// PrintStep prints a step header
func PrintStep(format string, args ...interface{}) {
	fmt.Println()
	colorStep.Printf("▶ "+format+"\n", args...)
	fmt.Println(separator())
}

// separator returns a visual separator
func separator() string {
	return "────────────────────────────────────────────────────────"
}

// getCurrentTimestamp returns the current timestamp as a string
func getCurrentTimestamp() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// CheckRoot checks if the program is running as root
func CheckRoot() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("this program must be run as root (use sudo)")
	}
	return nil
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsVMware checks if running on VMware
func IsVMware() (bool, error) {
	// Check DMI product name
	data, err := os.ReadFile("/sys/class/dmi/id/product_name")
	if err == nil {
		productName := string(data)
		if contains(productName, "VMware") {
			return true, nil
		}
	}

	// Check for VMware in /proc/cpuinfo
	data, err = os.ReadFile("/proc/cpuinfo")
	if err == nil {
		cpuInfo := string(data)
		if contains(cpuInfo, "VMware") || contains(cpuInfo, "hypervisor") {
			return true, nil
		}
	}

	// Check for vmware modules
	data, err = os.ReadFile("/proc/modules")
	if err == nil {
		modules := string(data)
		if contains(modules, "vmw_") || contains(modules, "vmxnet") {
			return true, nil
		}
	}

	return false, nil
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Banner prints the application banner
func Banner() {
	banner := `
╔══════════════════════════════════════════════════════════╗
║                                                          ║
║           VMware VM Performance Tuner                    ║
║                                                          ║
║  Optimize your VMware virtual machine for maximum        ║
║  performance with industry-standard best practices       ║
║                                                          ║
╚══════════════════════════════════════════════════════════╝
`
	colorStep.Println(banner)
}

// Summary prints a summary of what will be done
func Summary(modules []string) {
	PrintStep("Tuning Summary")
	fmt.Println("The following optimizations will be applied:")
	fmt.Println()

	for i, module := range modules {
		fmt.Printf("  %d. %s\n", i+1, module)
	}

	fmt.Println()
}

// CompletionMessage prints the completion message
func CompletionMessage(rebootRequired bool) {
	fmt.Println()
	colorSuccess.Println("╔══════════════════════════════════════════════════════════╗")
	colorSuccess.Println("║                                                          ║")
	colorSuccess.Println("║            Tuning Completed Successfully!                ║")
	colorSuccess.Println("║                                                          ║")
	colorSuccess.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()

	if rebootRequired {
		PrintWarning("IMPORTANT: A system reboot is required for all changes to take effect")
		PrintInfo("Run: sudo reboot")
		fmt.Println()
	}

	PrintInfo("Backup location: /root/.vmware-tuner-backups/")
	PrintInfo("To rollback changes, see the rollback.sh script in the backup directory")
	fmt.Println()
}
