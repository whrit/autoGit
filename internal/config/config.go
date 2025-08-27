package config

import (
    "bufio"
    "errors"
    "fmt"
    "io/fs"
    "os"
    "path/filepath"
    "strings"
    "time"

    "gopkg.in/yaml.v3"
)

type RepoConfig struct {
    Path         string        `yaml:"path"`
    Interval     time.Duration `yaml:"interval"`       // 0 disables timer
    Watch        bool          `yaml:"watch"`
    DebounceMS   int           `yaml:"debounce_ms"`    // debounce for fs events
    BatchWindow  time.Duration `yaml:"batch_window"`   // accumulate at least this long
    IdleWindow   time.Duration `yaml:"idle_window"`    // or fire when idle this long
    Push         bool          `yaml:"push"`
    Remote       string        `yaml:"remote"`
    Branch       string        `yaml:"branch"`
    Msg          string        `yaml:"msg"`
    Excludes     []string      `yaml:"excludes"`
    ParseIgnore  bool          `yaml:"parse_gitignore"`
    Sign         bool          `yaml:"sign"`
    SignArgs     []string      `yaml:"sign_args"`
    Trailers     map[string]string `yaml:"trailers"`
}

type Config struct {
    Theme      string `yaml:"theme"`
    LogPath    string `yaml:"log_path"`
    LogMaxSize int    `yaml:"log_max_size_mb"`
    LogMaxBackups int `yaml:"log_max_backups"`
    LogMaxAge  int    `yaml:"log_max_age_days"`

    InstallLaunchAgent bool `yaml:"install_launch_agent"`
    StartIntervalSec   int  `yaml:"start_interval_sec"`

    Repos []RepoConfig `yaml:"repos"`
}

// Defaults
func DefaultRepo(path string) RepoConfig {
    return RepoConfig{
        Path:        path,
        Interval:    0,
        Watch:       true,
        DebounceMS:  1200,
        BatchWindow: 45 * time.Second,
        IdleWindow:  5 * time.Second,
        Push:        false,
        Remote:      "origin",
        Branch:      "",
        Msg:         "autosave: {iso}",
        Excludes:    []string{"**/node_modules/**"},
        ParseIgnore: true,
        Sign:        false,
        SignArgs:    nil,
        Trailers:    map[string]string{},
    }
}

func Default() Config {
    home, _ := os.UserHomeDir()
    return Config{
        Theme:         "auto",
        LogPath:       filepath.Join(home, "Library", "Logs", "autoGit.log"),
        LogMaxSize:    10,
        LogMaxBackups: 3,
        LogMaxAge:     14,
        InstallLaunchAgent: false,
        StartIntervalSec:   0,
        Repos:        []RepoConfig{DefaultRepo(".")},
    }
}

// Config file helpers
func Path() string {
    if p := os.Getenv("GITAUTOCOMMIT_CONFIG"); p != "" { return p }
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".config", "autoGit", "config.yaml")
}

func Load() (Config, bool, error) {
    p := Path()
    b, err := os.ReadFile(p)
    if err != nil {
        if errors.Is(err, os.ErrNotExist) { return Config{}, false, nil }
        return Config{}, false, err
    }
    var c Config
    if err := yaml.Unmarshal(b, &c); err != nil { return Config{}, false, err }
    return c, true, nil
}

func Save(c Config) error {
    b, err := yaml.Marshal(c)
    if err != nil { return err }
    if err := os.MkdirAll(filepath.Dir(Path()), 0o755); err != nil { return err }
    return os.WriteFile(Path(), b, 0o644)
}

// Wizard
func RunWizard(cfg Config) (Config, error) {
    fmt.Println("▶ autoGit setup wizard")
    in := bufio.NewScanner(os.Stdin)

    ask := func(prompt, def string) string {
        if def != "" { fmt.Printf("? %s [%s]: ", prompt, def) } else { fmt.Printf("? %s: ", prompt) }
        if !in.Scan() { return def }
        s := strings.TrimSpace(in.Text()); if s == "" { return def }; return s
    }

    cfg.Theme = firstNonEmpty(ask("Theme (auto|dark|light|mono)", cfg.Theme), "auto")
    if cfg.LogPath == "" { home, _ := os.UserHomeDir(); cfg.LogPath = filepath.Join(home, "Library", "Logs", "autoGit.log") }
    cfg.LogPath = firstNonEmpty(ask("Log file path", cfg.LogPath), cfg.LogPath)

    var repos []RepoConfig
    for {
        p := ask("Path to Git repo (blank to stop)", ".")
        if strings.TrimSpace(p) == "" { break }
        r := DefaultRepo(p)
        mode := strings.ToLower(ask("Mode: (watch/timer/both)", "both"))
        switch mode { case "watch": r.Watch, r.Interval = true, 0; case "timer": r.Watch, r.Interval = false, 20*time.Minute; default: r.Watch, r.Interval = true, 20*time.Minute }
        iv := ask("Timer interval (e.g., 20m, 1h) — 0 to disable", r.Interval.String()); if d, err := time.ParseDuration(iv); err == nil { r.Interval = d }
        r.Push = yesno(ask("Push after commit? (y/n)", ternStr(r.Push, "y", "n")))
        r.Remote = firstNonEmpty(ask("Remote name", r.Remote), "origin")
        r.Branch = ask("Branch to push (blank = current)", r.Branch)
        r.Msg = firstNonEmpty(ask("Commit message template", r.Msg), r.Msg)
        r.DebounceMS = atoiDefault(ask("Watch debounce (ms)", fmt.Sprintf("%d", r.DebounceMS)), r.DebounceMS)
        r.BatchWindow = parseDurDefault(ask("Batch window (e.g., 45s)", r.BatchWindow.String()), r.BatchWindow)
        r.IdleWindow = parseDurDefault(ask("Idle window (e.g., 5s)", r.IdleWindow.String()), r.IdleWindow)
        r.ParseIgnore = yesno(ask("Parse .gitignore? (y/n)", ternStr(r.ParseIgnore, "y", "n")))
        if yesno(ask("Enable signed commits (-S)? (y/n)", ternStr(r.Sign, "y", "n"))) { r.Sign = true }
        ex := ask("Exclude globs (comma-separated)", strings.Join(r.Excludes, ",")); if strings.TrimSpace(ex) != "" { r.Excludes = splitAndTrim(ex, ",") }
        repos = append(repos, r)
    }
    if len(repos) == 0 { repos = []RepoConfig{ DefaultRepo(".") } }
    cfg.Repos = repos

    if yesno(ask("Create & load a LaunchAgent now? (y/n)", "n")) {
        cfg.InstallLaunchAgent = true
        cfg.StartIntervalSec = atoiDefault(ask("StartInterval seconds (0 to omit)", fmt.Sprintf("%d", cfg.StartIntervalSec)), 0)
    }

    return cfg, nil
}

// Helpers
func firstNonEmpty(a ...string) string { for _, s := range a { if strings.TrimSpace(s) != "" { return s } } ; return "" }
func yesno(s string) bool { s = strings.ToLower(strings.TrimSpace(s)); return s == "y" || s == "yes" || s == "true" }
func ternStr(b bool, t, f string) string { if b { return t } ; return f }
func atoiDefault(s string, d int) int { var n int; if _, err := fmt.Sscanf(s, "%d", &n); err == nil { return n }; return d }
func parseDurDefault(s string, d time.Duration) time.Duration { if v, err := time.ParseDuration(s); err == nil { return v }; return d }
func IsTerminal() bool { fi, _ := os.Stdin.Stat(); return (fi.Mode() & fs.ModeCharDevice) != 0 }