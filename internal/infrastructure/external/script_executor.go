package external

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ScriptExecutor executes user-defined scripts
type ScriptExecutor struct {
	enabled    bool
	scriptsDir string
}

// NewScriptExecutor creates a new script executor
func NewScriptExecutor(enabled bool, scriptsDir string) *ScriptExecutor {
	return &ScriptExecutor{
		enabled:    enabled,
		scriptsDir: scriptsDir,
	}
}

// RunScript executes a script with the given environment variables
func (e *ScriptExecutor) RunScript(scriptPath string, env map[string]string) error {
	if !e.enabled {
		return fmt.Errorf("script execution is disabled")
	}

	// Resolve script path (support relative paths from scripts directory)
	fullPath := scriptPath
	if !filepath.IsAbs(scriptPath) {
		fullPath = filepath.Join(e.scriptsDir, scriptPath)
	}

	// Check if script exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fmt.Errorf("script not found: %s", fullPath)
	}

	// Check if script is executable
	info, err := os.Stat(fullPath)
	if err != nil {
		return fmt.Errorf("failed to stat script: %w", err)
	}

	// On Unix-like systems, check executable bit
	if info.Mode()&0111 == 0 {
		return fmt.Errorf("script is not executable: %s", fullPath)
	}

	// Prepare command
	cmd := exec.Command(fullPath)

	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Execute script
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("script execution failed: %w (output: %s)", err, string(output))
	}

	return nil
}

// ValidateScript checks if a script exists and is executable
func (e *ScriptExecutor) ValidateScript(scriptPath string) error {
	fullPath := scriptPath
	if !filepath.IsAbs(scriptPath) {
		fullPath = filepath.Join(e.scriptsDir, scriptPath)
	}

	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("script not found: %s", fullPath)
	}
	if err != nil {
		return fmt.Errorf("failed to stat script: %w", err)
	}

	if info.Mode()&0111 == 0 {
		return fmt.Errorf("script is not executable: %s", fullPath)
	}

	return nil
}
