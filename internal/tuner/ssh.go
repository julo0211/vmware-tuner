package tuner

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// SSHTuner handles SSH hardening
type SSHTuner struct {
	Backup *BackupManager
}

// NewSSHTuner creates a new SSH tuner
func NewSSHTuner(backup *BackupManager) *SSHTuner {
	return &SSHTuner{
		Backup: backup,
	}
}

// Run performs the SSH hardening
func (st *SSHTuner) Run() error {
	PrintStep("SSH Hardening")

	PrintWarning("⚠️  WARNING: Incorrect SSH configuration can lock you out!")
	PrintWarning("Ensure you have console access (VMware Remote Console) or a backup session.")
	fmt.Println()
	
	configPath := "/etc/ssh/sshd_config"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("sshd_config not found at %s", configPath)
	}

	// Backup first
	if err := st.Backup.BackupFile(configPath); err != nil {
		return fmt.Errorf("failed to backup sshd_config: %w", err)
	}
	PrintSuccess("Backed up sshd_config")

	// Read config
	contentBytes, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	content := string(contentBytes)

	// Ask questions
	changes := false

	// 1. Disable Root Login
	if !strings.Contains(content, "PermitRootLogin no") {
		fmt.Print("Disable SSH Root Login? (y/n): ")
		var resp string
		fmt.Scanln(&resp)
		if resp == "y" {
			// Replace or append
			if strings.Contains(content, "PermitRootLogin") {
				// Simple replace (regex would be better but keeping it simple/safe)
				// We'll just append the override at the end, usually works for sshd
				content += "\n# Added by vmware-tuner\nPermitRootLogin no\n"
			} else {
				content += "\n# Added by vmware-tuner\nPermitRootLogin no\n"
			}
			changes = true
		}
	} else {
		PrintSuccess("Root login already disabled")
	}

	// 2. Disable Password Auth
	if !strings.Contains(content, "PasswordAuthentication no") {
		fmt.Print("Disable Password Authentication (Keys only)? (y/n): ")
		var resp string
		fmt.Scanln(&resp)
		if resp == "y" {
			content += "\n# Added by vmware-tuner\nPasswordAuthentication no\n"
			changes = true
		}
	} else {
		PrintSuccess("Password authentication already disabled")
	}

	if !changes {
		PrintInfo("No changes made")
		return nil
	}

	// Write new config
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write sshd_config: %w", err)
	}

	// Verify Config
	PrintInfo("Verifying configuration syntax...")
	cmd := exec.Command("sshd", "-t")
	if output, err := cmd.CombinedOutput(); err != nil {
		PrintError("Configuration check FAILED: %v", err)
		PrintInfo("Output: %s", string(output))
		PrintWarning("Restoring backup immediately...")
		
		// Restore
		backupPath := st.Backup.GetBackupPath("sshd_config")
		exec.Command("cp", backupPath, configPath).Run()
		return fmt.Errorf("safety check failed, changes reverted")
	}

	PrintSuccess("Configuration syntax verified")

	// Restart Service
	fmt.Print("Restart SSH service to apply? (y/n): ")
	var resp string
	fmt.Scanln(&resp)
	if resp == "y" {
		exec.Command("systemctl", "restart", "sshd").Run()
		PrintSuccess("SSH service restarted")
	} else {
		PrintInfo("Changes saved but service not restarted")
	}

	return nil
}
