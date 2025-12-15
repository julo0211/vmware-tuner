package tuner

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// RunCommand executes a shell command and manages output
func RunCommand(name string, args ...string) error {
	PrintInfo("Running: %s %s", name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed: %w", err)
	}
	return nil
}

// RunCommandSilent executes a shell command without streaming output to stdout
// Returns output and error
func RunCommandSilent(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// AskUser prompts the user with a question and returns true for yes, false for no
func AskUser(question string) bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s (y/n): ", question)
		input, _ := reader.ReadString('\n')
		input = strings.ToLower(strings.TrimSpace(input))

		if input == "y" || input == "yes" {
			return true
		}
		if input == "n" || input == "no" {
			return false
		}
		PrintWarning("Please answer 'y' or 'n'")
	}
}

// Pause waits for the user to press Enter
func Pause() {
	fmt.Println()
	fmt.Println("Press Enter to return to menu...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
