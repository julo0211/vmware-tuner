package tuner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsVMware_Detection(t *testing.T) {
	// Create temporary directory to simulate /sys and /proc
	tempDir, err := os.MkdirTemp("", "vmware_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Simulate DMI product name
	dmiDir := filepath.Join(tempDir, "sys", "class", "dmi", "id")
	if err := os.MkdirAll(dmiDir, 0755); err != nil {
		t.Fatalf("Failed to create dmi dir: %v", err)
	}

	productPath := filepath.Join(dmiDir, "product_name")
	if err := os.WriteFile(productPath, []byte("VMware Virtual Platform"), 0644); err != nil {
		t.Fatalf("Failed to write product_name: %v", err)
	}

	// Test positive case
	isVM, err := IsVMware(tempDir)
	if err != nil {
		t.Errorf("IsVMware returned error: %v", err)
	}
	if !isVM {
		t.Error("IsVMware should return true when 'VMware' is in product_name")
	}

	// Test negative case (overwrite file)
	if err := os.WriteFile(productPath, []byte("Physical Machine"), 0644); err != nil {
		t.Fatalf("Failed to overwrite product_name: %v", err)
	}

	isVM, err = IsVMware(tempDir)
	if err != nil {
		t.Errorf("IsVMware returned error: %v", err)
	}
	if isVM {
		t.Error("IsVMware should return false when 'VMware' is NOT in product_name")
	}
}
