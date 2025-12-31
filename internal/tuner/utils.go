package tuner

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
)

var (
	colorSuccess = color.New(color.FgGreen, color.Bold)
	colorError   = color.New(color.FgRed, color.Bold)
	colorWarning = color.New(color.FgYellow, color.Bold)
	colorInfo    = color.New(color.FgCyan)
	colorStep    = color.New(color.FgMagenta, color.Bold)
)

func PrintSuccess(format string, args ...interface{}) {
	colorSuccess.Print("✓ ")
	fmt.Printf(format+"\n", args...)
}

func PrintError(format string, args ...interface{}) {
	colorError.Print("✗ ")
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func PrintWarning(format string, args ...interface{}) {
	colorWarning.Print("⚠ ")
	fmt.Printf(format+"\n", args...)
}

func PrintInfo(format string, args ...interface{}) {
	colorInfo.Print("ℹ ")
	fmt.Printf(format+"\n", args...)
}

func PrintStep(format string, args ...interface{}) {
	fmt.Println()
	colorStep.Printf("▶ "+format+"\n", args...)
	fmt.Println("────────────────────────────────────────────────────────")
}

// CheckConnectivity verifies internet access via HTTP HEAD requests
// It respects HTTP_PROXY/HTTPS_PROXY environment variables automatically
func CheckConnectivity() bool {
	// Use reliable, highly available repositories
	endpoints := []string{
		"http://deb.debian.org",
		"http://mirror.centos.org",
		"http://github.com",
	}

	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	for _, url := range endpoints {
		// HEAD is lightweight (headers only)
		resp, err := client.Head(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			return true
		}
	}

	return false
}

func CheckRoot() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("ce programme doit être lancé en root (sudo)")
	}
	return nil
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func IsVMware(fsRoot string) (bool, error) {
	// Check DMI product name
	dmiPath := filepath.Join(fsRoot, "/sys/class/dmi/id/product_name")
	data, err := os.ReadFile(dmiPath)
	if err == nil {
		if strings.Contains(string(data), "VMware") {
			return true, nil
		}
	}

	// Check /proc/cpuinfo
	cpuInfoPath := filepath.Join(fsRoot, "/proc/cpuinfo")
	data, err = os.ReadFile(cpuInfoPath)
	if err == nil {
		content := string(data)
		if strings.Contains(content, "VMware") || strings.Contains(content, "hypervisor") {
			return true, nil
		}
	}
	return false, nil
}

func Banner() {
	banner := `
╔══════════════════════════════════════════════════════════╗
║                                                          ║
║           VMware VM Performance Tuner                    ║
║                                                          ║
║   Optimisé pour Environnements Enterprise (Air-Gapped)   ║
║                                                          ║
╚══════════════════════════════════════════════════════════╝
`
	colorStep.Println(banner)
}

func Summary(modules []string) {
	PrintStep("Résumé des actions")
	fmt.Println("Les optimisations suivantes seront appliquées :")
	fmt.Println()
	for i, module := range modules {
		fmt.Printf("  %d. %s\n", i+1, module)
	}
	fmt.Println()
}

func CompletionMessage(rebootRequired bool) {
	fmt.Println()
	PrintSuccess("Opérations terminées avec succès.")
	if rebootRequired {
		PrintWarning("IMPORTANT : Un redémarrage est nécessaire.")
	}
	PrintInfo("Backups disponibles dans /root/.vmware-tuner-backups/")
}

// getCurrentTimestamp returns the current time needed by other modules
func getCurrentTimestamp() string {
	return time.Now().Format("20060102-150405")
}
