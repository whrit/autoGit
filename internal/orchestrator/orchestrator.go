package orchestrator

import (
	"log"
	"sync"
	"time"

	"github.com/whrit/autoGit/internal/config"
	"github.com/whrit/autoGit/internal/gitops"
	"github.com/whrit/autoGit/internal/theme"
	"github.com/whrit/autoGit/internal/watch"
)

// Run starts workers for all repos and blocks until they exit.
func Run(cfg config.Config, t theme.Theme) {
	var wg sync.WaitGroup
	for _, rc := range cfg.Repos {
		rc := rc
		wg.Add(1)
		go func() { defer wg.Done(); runRepo(rc, t) }()
	}
	wg.Wait()
}

func runRepo(rc config.RepoConfig, t theme.Theme) {
	if !gitops.IsGitRepo(rc.Path) {
		log.Printf("[WARN] not a git repo: %s", rc.Path)
		return
	}

	// Event stream
	var (
		changes <-chan string
		stop    func()
	)
	if rc.Watch {
		ch, st, err := watch.Start(rc)
		if err != nil {
			log.Printf("[ERROR] watch: %v", err)
		} else {
			changes, stop = ch, st
		}
	}

	// Timers
	var (
		ticker *time.Ticker
	)
	if rc.Interval > 0 {
		ticker = time.NewTicker(rc.Interval)
	}

	// Batching state
	set := map[string]struct{}{}
	var (
		batchTimer *time.Timer
		idleTimer  *time.Timer
		mu         sync.Mutex
	)

	flush := func(reason string) {
		mu.Lock()
		files := make([]string, 0, len(set))
		for f := range set {
			files = append(files, f)
		}
		set = map[string]struct{}{}
		if batchTimer != nil {
			batchTimer.Stop()
			batchTimer = nil
		}
		if idleTimer != nil {
			idleTimer.Stop()
			idleTimer = nil
		}
		mu.Unlock()

		if len(files) == 0 && rc.Interval == 0 {
			return
		}

		msg, err := gitops.CommitAndMaybePush(rc, files)
		if err != nil {
			log.Printf("[ERROR] commit (%s): %v", rc.Path, err)
			return
		}
		if msg != "" {
			log.Printf("[OK] committed (%s): %s", rc.Path, msg)
		}
	}

	startTimers := func() {
		if rc.BatchWindow > 0 && batchTimer == nil {
			batchTimer = time.AfterFunc(rc.BatchWindow, func() { flush("batch") })
		}
		if rc.IdleWindow > 0 {
			if idleTimer != nil {
				idleTimer.Stop()
			}
			idleTimer = time.AfterFunc(rc.IdleWindow, func() { flush("idle") })
		}
	}

	// Event loop
	for {
		select {
		case f, ok := <-changes:
			if !ok {
				if stop != nil {
					stop()
				}
				if ticker != nil {
					ticker.Stop()
				}
				flush("shutdown")
				return
			}
			mu.Lock()
			set[f] = struct{}{}
			mu.Unlock()
			startTimers()
		case <-func() <-chan time.Time {
			if ticker != nil {
				return ticker.C
			}
			return nil
		}():
			flush("interval")
		}
	}
}
