package gitops

import (
    "bytes"
    "fmt"
    "os/exec"
    "path/filepath"
    "sort"
    "strings"
    "time"

    "github.com/whrit/autoGit/internal/config"
)

func mustRun(dir, name string, args ...string) error {
    cmd := exec.Command(name, args...)
    cmd.Dir = dir
    var stderr bytes.Buffer
    cmd.Stderr &= stderr
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("%s %s: %w (%s)", name, strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
    }
    return nil
}

func runOut(dir, name string, args ...string) (string, error) {
    cmd := exec.Command(name, args...)
    cmd.Dir = dir
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr
    err := cmd.Run()
    return stdout.String(), err
}

func IsGitRepo(path string) bool {
    _, err := exec.LookPath("git")
    if err != nil { return false }
    if _, err := runOut(path, "git", "rev-parse", "--is-inside-work-tree"); err != nil { return false }
    return true
}

func HasChanges(repo string) bool {
    if err := mustRun(repo, "git", "diff", "--quiet"); err != nil { return true }
    if err := mustRun(repo, "git", "diff", "--cached", "--quiet"); err != nil { return true }
    out, _ := runOut(repo, "git", "ls-files", "--others", "--exclude-standard")
    return strings.TrimSpace(out) != ""
}

func CurrentBranch(repo string) string {
    out, _ := runOut(repo, "git", "rev-parse", "--abbrev-ref", "HEAD")
    return strings.TrimSpace(out)
}

func RenderMessage(tpl string, files []string, rc config.RepoConfig) string {
    now := time.Now()
    msg := strings.ReplaceAll(tpl, "{iso}", now.UTC().Format(time.RFC3339))
    msg = strings.ReplaceAll(msg, "{unix}", fmt.Sprintf("%d", now.Unix()))
    msg = strings.ReplaceAll(msg, "{branch}", firstNonEmpty(rc.Branch, CurrentBranch(rc.Path)))
    if len(files) > 0 { msg = strings.ReplaceAll(msg, "{file}", filepath.Base(files[0])) } else { msg = strings.ReplaceAll(msg, "{file}", "") }
    msg = strings.ReplaceAll(msg, "{count}", fmt.Sprintf("%d", len(files)))
    if strings.TrimSpace(msg) == "" { msg = "autosave" }
    return msg
}

func CommitAndMaybePush(rc config.RepoConfig, files []string) (string, error) {
    if !HasChanges(rc.Path) { return "", nil }

    if err := mustRun(rc.Path, "git", "add", "-A"); err != nil { return "", err }

    // trailers
    trailerLines := make([]string, 0, len(rc.Trailers))
    keys := make([]string, 0, len(rc.Trailers))
    for k := range rc.Trailers { keys = append(keys, k) }
    sort.Strings(keys)
    for _, k := range keys { trailerLines = append(trailerLines, fmt.Sprintf("%s: %s", k, rc.Trailers[k])) }

    msg := RenderMessage(rc.Msg, files, rc)
    if len(trailerLines) > 0 { msg = msg + "

" + strings.Join(trailerLines, "
") }

    args := []string{"commit", "-m", msg}
    if rc.Sign { args = append(args, "-S") }
    if len(rc.SignArgs) > 0 { args = append(args, rc.SignArgs...) }

    if err := mustRun(rc.Path, "git", args...); err != nil {
        // if nothing to commit, surface no error
        if strings.Contains(err.Error(), "nothing to commit") || strings.Contains(err.Error(), "exit status 1") {
            return "", nil
        }
        return "", err
    }

    if rc.Push {
        pushArgs := []string{"push", firstNonEmpty(rc.Remote, "origin")}
        if rc.Branch != "" { pushArgs = append(pushArgs, fmt.Sprintf("HEAD:%s", rc.Branch)) }
        if err := mustRun(rc.Path, "git", pushArgs...); err != nil { return msg, err }
    }

    return msg, nil
}

func firstNonEmpty(a ...string) string { for _, s := range a { if strings.TrimSpace(s) != "" { return s } } ; return "" }