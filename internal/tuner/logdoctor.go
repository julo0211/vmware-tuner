package tuner

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// LogDoctorTuner handles log scanning
type LogDoctorTuner struct {
	Distro *DistroManager
}

// NewLogDoctorTuner creates a new log doctor
func NewLogDoctorTuner(distro *DistroManager) *LogDoctorTuner {
	return &LogDoctorTuner{
		Distro: distro,
	}
}

// Run performs the log scan
func (ld *LogDoctorTuner) Run() error {
	PrintStep("Log Doctor (Troubleshoot)")

	keywords := []string{
		"Out of memory",
		"Kill process",
		"I/O error",
		"SCSI error",
		"Call Trace",
		"soft lockup",
		"segfault",
		"EXT4-fs error",
		"XFS_WANT_CORRUPT",
	}

	foundIssues := false

	// 1. Check dmesg (Kernel Ring Buffer)
	PrintInfo("Scanning kernel ring buffer (dmesg)...")
	out, err := exec.Command("dmesg").Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		// Check last 1000 lines to avoid noise from boot time if uptime is long
		start := 0
		if len(lines) > 1000 {
			start = len(lines) - 1000
		}
		
		for i := start; i < len(lines); i++ {
			line := lines[i]
			for _, kw := range keywords {
				if strings.Contains(line, kw) {
					PrintWarning("Found in dmesg: %s", line)
					foundIssues = true
				}
			}
		}
	}

	// 2. Check System Log
	logFile := "/var/log/syslog"
	if ld.Distro.Type == DistroRHEL {
		logFile = "/var/log/messages"
	}

	PrintInfo("Scanning system log (%s)...", logFile)
	if _, err := os.Stat(logFile); err == nil {
		// Use grep for efficiency
		for _, kw := range keywords {
			// grep -i "keyword" /var/log/syslog | tail -n 5
			cmd := exec.Command("bash", "-c", fmt.Sprintf("grep -i \"%s\" %s | tail -n 5", kw, logFile))
			out, err := cmd.Output()
			if err == nil && len(out) > 0 {
				PrintWarning("Found '%s' errors:", kw)
				fmt.Println(string(out))
				foundIssues = true
			}
		}
	} else {
		PrintInfo("Log file not found: %s", logFile)
	}

	if !foundIssues {
		PrintSuccess("No critical errors found in recent logs.")
	} else {
		fmt.Println()
		PrintInfo("Issues were found. Please investigate the logs further.")
	}

	return nil
}
