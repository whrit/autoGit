package agent

import (
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"

    "github.com/whrit/autoGit/internal/config"
)

func WriteAndLoadLaunchAgent(cfg config.Config, exe string) error {
    home, _ := os.UserHomeDir()
    plistPath := filepath.Join(home, "Library", "LaunchAgents", "com.gitautocommit.cli.plist")

    var b strings.Builder
    b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>
")
    b.WriteString("<!DOCTYPE plist PUBLIC \"-//Apple//DTD PLIST 1.0//EN\" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\">
")
    b.WriteString("<plist version=\"1.0\"><dict>
")
    b.WriteString("  <key>Label</key><string>com.gitautocommit.cli</string>
")
    b.WriteString("  <key>ProgramArguments</key><array>
")
    b.WriteString("    <string>" + exe + "</string>
")
    b.WriteString("    <string>--theme</string><string>" + cfg.Theme + "</string>
")
    b.WriteString("  </array>
")
    b.WriteString("  <key>RunAtLoad</key><true/>
")
    if cfg.StartIntervalSec > 0 { b.WriteString(fmt.Sprintf("  <key>StartInterval</key><integer>%d</integer>
", cfg.StartIntervalSec)) }
    b.WriteString("  <key>StandardOutPath</key><string>/tmp/autoGit.out</string>
")
    b.WriteString("  <key>StandardErrorPath</key><string>/tmp/autoGit.err</string>
")
    b.WriteString("</dict></plist>
")

    if err := os.WriteFile(plistPath, []byte(b.String()), 0o644); err != nil { return err }
    if err := exec.Command("launchctl", "load", plistPath).Run(); err != nil {
        return fmt.Errorf("launchctl load: %w", err)
    }
    return nil
}