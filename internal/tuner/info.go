package tuner

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// InfoTuner handles system info display
type InfoTuner struct{}

// NewInfoTuner creates a new info tuner
func NewInfoTuner() *InfoTuner {
	return &InfoTuner{}
}

// Run displays the info
func (it *InfoTuner) Run() error {
	PrintStep("System Information")

	// 1. OS Info
	osInfo := "Unknown"
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				osInfo = strings.Trim(line[12:], "\"")
				break
			}
		}
	}
	fmt.Printf("  %-20s: %s\n", "OS", osInfo)

	// 2. Kernel
	if out, err := exec.Command("uname", "-r").Output(); err == nil {
		fmt.Printf("  %-20s: %s", "Kernel", string(out))
	}

	// 3. CPU
	// grep -c processor /proc/cpuinfo
	if out, err := exec.Command("bash", "-c", "grep -c processor /proc/cpuinfo").Output(); err == nil {
		fmt.Printf("  %-20s: %s", "vCPUs", string(out))
	}

	// 4. Memory
	// free -h | grep Mem | awk '{print $2}'
	if out, err := exec.Command("bash", "-c", "free -h | grep Mem").Output(); err == nil {
		parts := strings.Fields(string(out))
		if len(parts) >= 3 {
			fmt.Printf("  %-20s: %s (Used: %s)\n", "Memory", parts[1], parts[2])
		}
	}

	// 5. IP Address
	// hostname -I | awk '{print $1}'
	if out, err := exec.Command("hostname", "-I").Output(); err == nil {
		ips := strings.TrimSpace(string(out))
		firstIp := strings.Split(ips, " ")[0]
		fmt.Printf("  %-20s: %s\n", "IP Address", firstIp)
	}

	// 6. VM Tools Status
	fmt.Printf("  %-20s: ", "VMware Tools")
	if err := exec.Command("systemctl", "is-active", "vmtoolsd").Run(); err == nil {
		PrintSuccess("Running")
	} else {
		PrintWarning("Not Running")
	}

	return nil
}
