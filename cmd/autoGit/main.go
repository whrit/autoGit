package main

import (
    "flag"
    "fmt"
    "log"
    "os"
    "path/filepath"

    "github.com/whrit/autoGit/internal/agent"
    "github.com/whrit/autoGit/internal/config"
    "github.com/whrit/autoGit/internal/logs"
    "github.com/whrit/autoGit/internal/orchestrator"
    "github.com/whrit/autoGit/internal/theme"
)

var (
    version = "v3.0.0"
)

func main() {
    var (
        setup  bool
        setTheme string
        addRepo string
    )

    flag.BoolVar(&setup, "setup", false, "Run interactive setup wizard and save config")
    flag.StringVar(&setTheme, "theme", "", "Override theme: auto|dark|light|mono")
    flag.StringVar(&addRepo, "add-repo", "", "Quick-add a repo path to config and save")
    flag.Parse()

    cfgPath := config.Path()
    cfg, found, err := config.Load()
    if err != nil { log.Fatalf("config: %v", err) }
    if !found { cfg = config.Default() }

    if setTheme != "" {
        cfg.Theme = setTheme
    }

    if addRepo != "" {
        cfg.Repos = append(cfg.Repos, config.DefaultRepo(addRepo))
        if err := config.Save(cfg); err != nil { log.Fatalf("save config: %v", err) }
        fmt.Printf("Added repo and saved config → %s
", cfgPath)
        return
    }

    if setup || !found {
        // Wizard
        if config.IsTerminal() {
            var werr error
            cfg, werr = config.RunWizard(cfg)
            if werr != nil { log.Fatalf("setup aborted: %v", werr) }
            if err := config.Save(cfg); err != nil { log.Fatalf("save config: %v", err) }
            fmt.Printf("Saved config → %s
", cfgPath)
            if cfg.InstallLaunchAgent {
                exe, _ := os.Executable()
                if err := agent.WriteAndLoadLaunchAgent(cfg, exe); err != nil {
                    log.Fatalf("launch agent: %v", err)
                }
            }
        } else {
            fmt.Println("--setup requested but no TTY; run in a terminal")
        }
    }

    // Bootstrap logging & theming
    if err := logs.Setup(cfg); err != nil { log.Fatalf("log setup: %v", err) }
    t := theme.FromName(cfg.Theme)

    log.Printf("autoGit %s starting in %s
", version, filepath.Dir(cfgPath))

    // Run orchestrator (blocks until workers complete)
    orchestrator.Run(cfg, t)
}