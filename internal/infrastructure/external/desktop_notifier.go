package external

import (
	"fmt"
	"os/exec"
	"runtime"
)

// DesktopNotifier sends desktop notifications
type DesktopNotifier struct {
	enabled bool
}

// NewDesktopNotifier creates a new desktop notifier
func NewDesktopNotifier(enabled bool) *DesktopNotifier {
	return &DesktopNotifier{
		enabled: enabled,
	}
}

// SendNotification sends a desktop notification
func (n *DesktopNotifier) SendNotification(title, message string, metadata map[string]string) error {
	if !n.enabled {
		return nil
	}

	switch runtime.GOOS {
	case "linux":
		return n.sendLinuxNotification(title, message)
	case "darwin":
		return n.sendMacOSNotification(title, message)
	case "windows":
		return n.sendWindowsNotification(title, message)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// sendLinuxNotification sends a notification on Linux using notify-send
func (n *DesktopNotifier) sendLinuxNotification(title, message string) error {
	// Check if notify-send is available
	if _, err := exec.LookPath("notify-send"); err != nil {
		return fmt.Errorf("notify-send not found: %w", err)
	}

	cmd := exec.Command("notify-send", "-u", "normal", "-a", "mkanban", title, message)
	return cmd.Run()
}

// sendMacOSNotification sends a notification on macOS using osascript
func (n *DesktopNotifier) sendMacOSNotification(title, message string) error {
	script := fmt.Sprintf(`display notification "%s" with title "%s" sound name "default"`,
		escapeAppleScript(message),
		escapeAppleScript(title))

	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

// sendWindowsNotification sends a notification on Windows using PowerShell
func (n *DesktopNotifier) sendWindowsNotification(title, message string) error {
	// Use PowerShell to show a notification
	script := fmt.Sprintf(`
		[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
		[Windows.UI.Notifications.ToastNotification, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
		[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null

		$template = @"
		<toast>
			<visual>
				<binding template="ToastText02">
					<text id="1">%s</text>
					<text id="2">%s</text>
				</binding>
			</visual>
		</toast>
		"@

		$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
		$xml.LoadXml($template)
		$toast = New-Object Windows.UI.Notifications.ToastNotification $xml
		[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("mkanban").Show($toast)
	`, escapeString(title), escapeString(message))

	cmd := exec.Command("powershell", "-Command", script)
	return cmd.Run()
}

// escapeAppleScript escapes special characters for AppleScript
func escapeAppleScript(s string) string {
	// Escape quotes and backslashes
	result := ""
	for _, c := range s {
		if c == '"' || c == '\\' {
			result += "\\"
		}
		result += string(c)
	}
	return result
}

// escapeString escapes special characters for PowerShell
func escapeString(s string) string {
	// Simple escape for PowerShell strings
	result := ""
	for _, c := range s {
		if c == '"' {
			result += "`"
		}
		result += string(c)
	}
	return result
}
