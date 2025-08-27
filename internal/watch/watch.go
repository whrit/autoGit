package watch

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/whrit/autoGit/internal/config"
)

// Start begins watching a repo and returns a channel of changed file paths and a stop func.
func Start(rc config.RepoConfig) (<-chan string, func(), error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, nil, err
	}

	changes := make(chan string, 128)
	stop := func() { _ = w.Close(); close(changes) }

	// merge excludes with .gitignore patterns (basic)
	excludes := append([]string{}, rc.Excludes...)
	if rc.ParseIgnore {
		excludes = append(excludes, readGitignore(rc.Path)...)
	}

	addDir := func(path string) error {
		base := filepath.Base(path)
		if strings.HasPrefix(base, ".git") {
			return filepath.SkipDir
		}
		if shouldExclude(path, rc.Path, excludes) {
			return filepath.SkipDir
		}
		return w.Add(path)
	}

	if err := filepath.WalkDir(rc.Path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return addDir(path)
		}
		return nil
	}); err != nil {
		_ = w.Close()
		return nil, nil, err
	}

	// debounce to coalesce flurries of events
	debounce := time.Duration(rc.DebounceMS) * time.Millisecond
	var mu sync.Mutex
	var timer *time.Timer
	var last string

	fire := func() {
		mu.Lock()
		f := last
		last = ""
		mu.Unlock()
		if f != "" {
			changes <- f
		}
	}

	reset := func(f string) {
		mu.Lock()
		last = f
		mu.Unlock()
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(debounce, fire)
	}

	go func() {
		defer stop()
		for {
			select {
			case ev, ok := <-w.Events:
				if !ok {
					return
				}
				if strings.Contains(ev.Name, string(os.PathSeparator)+".git"+string(os.PathSeparator)) {
					continue
				}
				if shouldExclude(ev.Name, rc.Path, excludes) {
					continue
				}
				if ev.Op&fsnotify.Create == fsnotify.Create {
					if fi, err := os.Stat(ev.Name); err == nil && fi.IsDir() {
						_ = addDir(ev.Name)
					}
				}
				reset(ev.Name)
			case err := <-w.Errors:
				_ = err // ignore but keep loop; stop() will be deferred on return
				return
			}
		}
	}()

	return changes, stop, nil
}

func readGitignore(root string) []string {
	f, err := os.Open(filepath.Join(root, ".gitignore"))
	if err != nil {
		return nil
	}
	defer f.Close()
	var pats []string
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		pats = append(pats, line)
	}
	return pats
}

func shouldExclude(p, root string, globs []string) bool {
	rel, _ := filepath.Rel(root, p)
	for _, g := range globs {
		if matchGlob(g, rel) {
			return true
		}
	}
	return false
}

func matchGlob(pattern, s string) bool {
	if ok, _ := filepath.Match(pattern, s); ok {
		return true
	}
	if !strings.Contains(pattern, "**") {
		return false
	}
	parts := strings.Split(pattern, "**")
	if len(parts) == 2 {
		pre := strings.TrimSuffix(parts[0], string(os.PathSeparator))
		suf := strings.TrimPrefix(parts[1], string(os.PathSeparator))
		return strings.HasPrefix(s, pre) && strings.HasSuffix(s, suf)
	}
	return false
}
