package tuner

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// DockerTuner handles Docker optimization
type DockerTuner struct{}

// NewDockerTuner creates a new docker tuner
func NewDockerTuner() *DockerTuner {
	return &DockerTuner{}
}

// Run performs the optimization
func (dt *DockerTuner) Run() error {
	PrintStep("Docker Optimizer")

	// 1. Check if Docker is installed
	if _, err := exec.LookPath("docker"); err != nil {
		PrintWarning("Docker is not installed.")
		return nil
	}
	PrintSuccess("Docker is installed")

	// 2. Check Log Rotation
	daemonFile := "/etc/docker/daemon.json"
	needsRotation := true

	if _, err := os.Stat(daemonFile); err == nil {
		content, _ := os.ReadFile(daemonFile)
		if strings.Contains(string(content), "log-driver") && strings.Contains(string(content), "max-size") {
			PrintSuccess("Log rotation is already configured")
			needsRotation = false
		}
	}

	if needsRotation {
		PrintWarning("Docker log rotation is NOT configured.")
		PrintInfo("Containers can fill the disk with logs.")
		fmt.Print("Configure log rotation (max-size=10m, max-file=3)? (y/n): ")
		var resp string
		fmt.Scanln(&resp)
		if resp == "y" {
			// Create or update daemon.json
			// Simple overwrite if not exists, or append warning if complex
			if _, err := os.Stat(daemonFile); os.IsNotExist(err) {
				content := `{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  }
}
`
				if err := os.WriteFile(daemonFile, []byte(content), 0644); err != nil {
					PrintWarning("Failed to write daemon.json: %v", err)
				} else {
					PrintSuccess("Configuration created. Restart Docker to apply.")
					exec.Command("systemctl", "restart", "docker").Run()
				}
			} else {
				PrintWarning("daemon.json exists. Please add log-opts manually to avoid overwriting custom config.")
			}
		}
	}

	// 3. Prune
	fmt.Println()
	PrintInfo("Docker System Prune")
	PrintInfo("This will remove:")
	PrintInfo("  - Stopped containers")
	PrintInfo("  - Unused networks")
	PrintInfo("  - Dangling images")
	PrintInfo("  - Build cache")
	fmt.Print("Run prune? (y/n): ")
	var resp string
	fmt.Scanln(&resp)
	if resp == "y" {
		cmd := exec.Command("docker", "system", "prune", "-f")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			PrintWarning("Prune failed: %v", err)
		} else {
			PrintSuccess("System pruned")
		}
	}

	return nil
}
