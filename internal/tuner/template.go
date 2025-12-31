package tuner

import (
	"fmt"
	"os"
	"os/exec"
)

// TemplateTuner handles VM sealing
type TemplateTuner struct{}

// NewTemplateTuner creates a new template tuner
func NewTemplateTuner() *TemplateTuner {
	return &TemplateTuner{}
}

// Run performs the sealing process
func (tt *TemplateTuner) Run() error {
	PrintStep("Seal VM for Template")

	PrintWarning("⚠️  DANGER ZONE ⚠️")
	PrintWarning("This will remove unique system identifiers (Machine ID, SSH Keys, Logs).")
	PrintWarning("The VM will be shut down immediately after.")
	PrintWarning("DO NOT RUN THIS if you are not creating a template/golden image.")
	fmt.Println()
	
	fmt.Print("Type 'SEAL' to continue: ")
	var response string
	fmt.Scanln(&response)
	
	if response != "SEAL" {
		PrintInfo("Operation cancelled (Safety check failed)")
		return nil
	}

	PrintInfo("Preparing system for templating...")

	// 1. Clean Machine ID
	// /etc/machine-id should be empty, not missing, for systemd to regenerate it
	PrintInfo("Resetting Machine ID...")
	if err := os.Truncate("/etc/machine-id", 0); err != nil {
		PrintWarning("Failed to truncate /etc/machine-id: %v", err)
	}
	os.Remove("/var/lib/dbus/machine-id")

	// 2. Remove SSH Host Keys
	PrintInfo("Removing SSH Host Keys...")
	exec.Command("rm", "-f", "/etc/ssh/ssh_host_*").Run()

	// 3. Clean Logs
	PrintInfo("Vacuuming logs...")
	exec.Command("journalctl", "--vacuum-time=1s").Run()
	exec.Command("rm", "-f", "/var/log/*.gz").Run()
	exec.Command("rm", "-f", "/var/log/*.[0-9]").Run()

	// 4. Clean Bash History
	PrintInfo("Clearing shell history...")
	os.Remove("/root/.bash_history")
	exec.Command("history", "-c").Run()

	// 5. Clean Package Cache (Reuse logic if possible, but simple command here is fine)
	PrintInfo("Cleaning package cache...")
	exec.Command("apt-get", "clean").Run()
	exec.Command("yum", "clean", "all").Run()

	PrintSuccess("System sealed successfully!")
	PrintInfo("Shutting down in 3 seconds...")
	
	exec.Command("sleep", "3").Run()
	exec.Command("poweroff").Run()

	return nil
}
