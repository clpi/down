package sync

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const maxDiffLines = 500

type GitIndex struct {
	Version     int      `json:"version"`
	RepoRoot    string   `json:"repo_root"`
	Branch      string   `json:"branch"`
	HEAD        string   `json:"head"`
	Commits     []string `json:"commits"`
	SyncedAt    string   `json:"synced_at"`
	CommitCount int      `json:"commit_count"`
}

type GitSyncResult struct {
	GitDir      string
	HEAD        string
	Branch      string
	Written     int
	Skipped     int
	WorkingTree bool
}

type commitInfo struct {
	Full    string
	Short   string
	Date    string
	Author  string
	Email   string
	Subject string
}

func gitRootDir(start string) (string, error) {
	out, err := runGit(start, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), strings.TrimSpace(string(out)), err)
	}
	return string(out), nil
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

func ensureGitDir(downDir string) string {
	gitDir := filepath.Join(downDir, "git")
	for _, sub := range []string{"commits", "branches", "diffs"} {
		os.MkdirAll(filepath.Join(gitDir, sub), 0755)
	}
	return gitDir
}

func loadGitIndex(gitDir string) *GitIndex {
	path := filepath.Join(gitDir, "index.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return &GitIndex{Version: 1, Commits: []string{}}
	}
	var idx GitIndex
	if json.Unmarshal(data, &idx) == nil {
		if idx.Commits == nil {
			idx.Commits = []string{}
		}
		return &idx
	}
	return &GitIndex{Version: 1, Commits: []string{}}
}

func saveGitIndex(gitDir string, idx *GitIndex) error {
	idx.SyncedAt = time.Now().Format(time.RFC3339)
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(gitDir, "index.json"), data, 0644)
}

func listCommits(repoRoot string) ([]commitInfo, error) {
	out, err := runGit(repoRoot, "log", "--reverse", "--format=%H|%h|%aI|%an|%ae|%s")
	if err != nil {
		return nil, err
	}
	var commits []commitInfo
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 6)
		if len(parts) < 6 {
			continue
		}
		commits = append(commits, commitInfo{
			Full:    parts[0],
			Short:   parts[1],
			Date:    parts[2],
			Author:  parts[3],
			Email:   parts[4],
			Subject: parts[5],
		})
	}
	return commits, nil
}

func commitStats(repoRoot, sha string) (added, removed, files int) {
	out, err := runGit(repoRoot, "show", "--stat", "--format=", sha)
	if err != nil {
		return 0, 0, 0
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 0 {
		return 0, 0, 0
	}
	summary := lines[len(lines)-1]
	fmt.Sscanf(summary, " %d file", &files)
	if strings.Contains(summary, "insertion") {
		fmt.Sscanf(summary, "%*s %d insertion", &added)
	}
	if strings.Contains(summary, "deletion") {
		fmt.Sscanf(summary, "%*s %*s %d deletion", &removed)
	}
	return added, removed, files
}

func commitPatch(repoRoot, sha string) string {
	out, err := runGit(repoRoot, "show", "--format=", "--no-color", sha)
	if err != nil {
		return ""
	}
	lines := strings.Split(out, "\n")
	if len(lines) > maxDiffLines {
		truncated := strings.Join(lines[:maxDiffLines], "\n")
		return truncated + fmt.Sprintf("\n\n... diff truncated (%d lines, showing %d) ...\n", len(lines), maxDiffLines)
	}
	return out
}

func commitBody(repoRoot, sha string) string {
	out, err := runGit(repoRoot, "log", "-1", "--format=%B", sha)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

func writeCommitFile(gitDir, repoRoot string, c commitInfo, force bool) (bool, error) {
	path := filepath.Join(gitDir, "commits", shortSHA(c.Full)+".md")
	if !force {
		if _, err := os.Stat(path); err == nil {
			return false, nil
		}
	}

	added, removed, files := commitStats(repoRoot, c.Full)
	body := commitBody(repoRoot, c.Full)
	patch := commitPatch(repoRoot, c.Full)

	var b strings.Builder
	fmt.Fprintf(&b, "# %s — %s\n\n", c.Short, c.Subject)
	fmt.Fprintf(&b, "> **SHA:** `%s`  \n", c.Full)
	fmt.Fprintf(&b, "> **Date:** %s  \n", c.Date)
	fmt.Fprintf(&b, "> **Author:** %s <%s>  \n", c.Author, c.Email)
	if files > 0 {
		fmt.Fprintf(&b, "> **Changes:** +%d −%d (%d files)\n\n", added, removed, files)
	} else {
		fmt.Fprintf(&b, "\n")
	}

	if body != "" && body != c.Subject {
		fmt.Fprintf(&b, "## Message\n\n%s\n\n", body)
	}

	if patch != "" {
		fmt.Fprintf(&b, "## Diff\n\n```diff\n%s```\n", patch)
	}

	if err := os.WriteFile(path, []byte(b.String()), 0644); err != nil {
		return false, err
	}
	return true, nil
}

func writeHistoryMarkdown(gitDir, repoRoot, branch, head string, commits []commitInfo) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# Git History\n\n")
	fmt.Fprintf(&b, "> **Repository:** %s  \n", repoRoot)
	fmt.Fprintf(&b, "> **Branch:** %s  \n", branch)
	fmt.Fprintf(&b, "> **HEAD:** `%s`  \n", head)
	fmt.Fprintf(&b, "> **Commits:** %d  \n", len(commits))
	fmt.Fprintf(&b, "> **Synced:** %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	currentMonth := ""
	for _, c := range commits {
		month := c.Date[:7]
		if month != currentMonth {
			if currentMonth != "" {
				b.WriteString("\n")
			}
			fmt.Fprintf(&b, "## %s\n\n", month)
			fmt.Fprintf(&b, "| SHA | Date | Author | Subject | Δ |\n")
			fmt.Fprintf(&b, "|-----|------|--------|---------|---|\n")
			currentMonth = month
		}
		added, removed, files := commitStats(repoRoot, c.Full)
		delta := ""
		if files > 0 {
			delta = fmt.Sprintf("+%d −%d", added, removed)
		}
		subject := strings.ReplaceAll(c.Subject, "|", "\\|")
		fmt.Fprintf(&b, "| [`%s`](commits/%s.md) | %s | %s | %s | %s |\n",
			c.Short, shortSHA(c.Full), c.Date[:10], c.Author, subject, delta)
	}

	return os.WriteFile(filepath.Join(gitDir, "history.md"), []byte(b.String()), 0644)
}

func writeBranchMarkdown(gitDir, branch string, commits []commitInfo) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# Branch: %s\n\n", branch)
	fmt.Fprintf(&b, "> %d commits (newest first)\n\n", len(commits))

	for i := len(commits) - 1; i >= 0; i-- {
		c := commits[i]
		fmt.Fprintf(&b, "- [`%s`](commits/%s.md) %s — %s (%s)\n",
			c.Short, shortSHA(c.Full), c.Date[:10], c.Subject, c.Author)
	}

	safeBranch := strings.ReplaceAll(branch, "/", "_")
	return os.WriteFile(filepath.Join(gitDir, "branches", safeBranch+".md"), []byte(b.String()), 0644)
}

func writeHEADMarkdown(gitDir, repoRoot, head string) error {
	commits, err := listCommits(repoRoot)
	if err != nil || len(commits) == 0 {
		return err
	}
	var headCommit *commitInfo
	for i := range commits {
		if commits[i].Full == head {
			headCommit = &commits[i]
			break
		}
	}
	if headCommit == nil {
		out, err := runGit(repoRoot, "log", "-1", "--format=%H|%h|%aI|%an|%ae|%s", head)
		if err != nil {
			return err
		}
		parts := strings.SplitN(strings.TrimSpace(out), "|", 6)
		if len(parts) >= 6 {
			headCommit = &commitInfo{
				Full: parts[0], Short: parts[1], Date: parts[2],
				Author: parts[3], Email: parts[4], Subject: parts[5],
			}
		}
	}
	if headCommit == nil {
		return nil
	}

	added, removed, files := commitStats(repoRoot, head)
	body := commitBody(repoRoot, head)
	patch := commitPatch(repoRoot, head)

	var b strings.Builder
	fmt.Fprintf(&b, "# HEAD: %s\n\n", headCommit.Subject)
	fmt.Fprintf(&b, "> **SHA:** `%s`  \n", head)
	fmt.Fprintf(&b, "> **Date:** %s  \n", headCommit.Date)
	fmt.Fprintf(&b, "> **Author:** %s <%s>  \n", headCommit.Author, headCommit.Email)
	if files > 0 {
		fmt.Fprintf(&b, "> **Changes:** +%d −%d (%d files)\n\n", added, removed, files)
	}

	if body != "" {
		fmt.Fprintf(&b, "## Message\n\n%s\n\n", body)
	}
	if patch != "" {
		fmt.Fprintf(&b, "## Diff\n\n```diff\n%s```\n", patch)
	}

	return os.WriteFile(filepath.Join(gitDir, "HEAD.md"), []byte(b.String()), 0644)
}

func writeWorkingTree(gitDir, repoRoot string) (bool, error) {
	diff, err := gitWorkingDiff(repoRoot)
	if err != nil {
		return false, err
	}
	path := filepath.Join(gitDir, "working-tree.md")
	if diff == "" {
		content := "# Working Tree\n\n> No uncommitted changes.\n"
		return false, os.WriteFile(path, []byte(content), 0644)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Working Tree\n\n")
	fmt.Fprintf(&b, "> Uncommitted changes as of %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "## Diff\n\n```diff\n%s```\n", diff)
	return true, os.WriteFile(path, []byte(b.String()), 0644)
}

func gitWorkingDiff(repoRoot string) (string, error) {
	out, err := runGit(repoRoot, "diff", "--no-color")
	if err != nil {
		return "", err
	}
	staged, err := runGit(repoRoot, "diff", "--cached", "--no-color")
	if err != nil {
		return "", err
	}
	var parts []string
	if strings.TrimSpace(staged) != "" {
		parts = append(parts, "## Staged\n\n"+staged)
	}
	if strings.TrimSpace(out) != "" {
		parts = append(parts, "## Unstaged\n\n"+out)
	}
	return strings.Join(parts, "\n\n"), nil
}

func syncGitHistory(downDir, repoRoot string, force, verbose bool) (*GitSyncResult, error) {
	gitDir := ensureGitDir(downDir)
	idx := loadGitIndex(gitDir)

	head, err := runGit(repoRoot, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	head = strings.TrimSpace(head)

	branch, err := runGit(repoRoot, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		branch = "unknown"
	} else {
		branch = strings.TrimSpace(branch)
	}

	commits, err := listCommits(repoRoot)
	if err != nil {
		return nil, err
	}

	result := &GitSyncResult{
		GitDir: gitDir,
		HEAD:   head,
		Branch: branch,
	}

	for _, c := range commits {
		written, err := writeCommitFile(gitDir, repoRoot, c, force)
		if err != nil {
			return nil, err
		}
		if written {
			result.Written++
			if verbose {
				fmt.Printf("  + commits/%s.md\n", shortSHA(c.Full))
			}
		} else {
			result.Skipped++
		}
	}

	if err := writeHistoryMarkdown(gitDir, repoRoot, branch, head, commits); err != nil {
		return nil, err
	}
	if err := writeBranchMarkdown(gitDir, branch, commits); err != nil {
		return nil, err
	}
	if err := writeHEADMarkdown(gitDir, repoRoot, head); err != nil {
		return nil, err
	}

	hasWorking, err := writeWorkingTree(gitDir, repoRoot)
	if err != nil {
		return nil, err
	}
	result.WorkingTree = hasWorking

	allSHAs := make([]string, len(commits))
	for i, c := range commits {
		allSHAs[i] = c.Full
	}
	idx.RepoRoot = repoRoot
	idx.Branch = branch
	idx.HEAD = head
	idx.Commits = allSHAs
	idx.CommitCount = len(commits)
	if err := saveGitIndex(gitDir, idx); err != nil {
		return nil, err
	}

	return result, nil
}

func gitStatusReport(downDir, repoRoot string) (string, error) {
	idx := loadGitIndex(filepath.Join(downDir, "git"))

	head, err := runGit(repoRoot, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	head = strings.TrimSpace(head)

	branch, _ := runGit(repoRoot, "rev-parse", "--abbrev-ref", "HEAD")
	branch = strings.TrimSpace(branch)

	var b strings.Builder
	fmt.Fprintf(&b, "Repository: %s\n", repoRoot)
	fmt.Fprintf(&b, "Branch:     %s\n", branch)
	fmt.Fprintf(&b, "HEAD:       %s\n", shortSHA(head))

	if idx.HEAD == "" {
		fmt.Fprintf(&b, "Sync:       never synced — run `down sync git`\n")
	} else if idx.HEAD != head {
		fmt.Fprintf(&b, "Sync:       stale (indexed %s, HEAD is %s)\n", shortSHA(idx.HEAD), shortSHA(head))
		fmt.Fprintf(&b, "            run `down sync git` to update .down/git/\n")
	} else {
		fmt.Fprintf(&b, "Sync:       up to date (%s, %d commits)\n", idx.SyncedAt, idx.CommitCount)
	}

	porcelain, _ := runGit(repoRoot, "status", "--porcelain")
	if strings.TrimSpace(porcelain) != "" {
		fmt.Fprintf(&b, "Working tree: dirty (see .down/git/working-tree.md after sync)\n")
	} else {
		fmt.Fprintf(&b, "Working tree: clean\n")
	}

	return b.String(), nil
}

func gitLogOutput(downDir, repoRoot string, limit int) (string, error) {
	commits, err := listCommits(repoRoot)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	start := len(commits) - limit
	if start < 0 {
		start = 0
	}
	for i := len(commits) - 1; i >= start; i-- {
		c := commits[i]
		fmt.Fprintf(&b, "%s %s %s — %s\n", c.Short, c.Date[:10], c.Author, c.Subject)
		fmt.Fprintf(&b, "  → .down/git/commits/%s.md\n", shortSHA(c.Full))
	}
	if len(commits) == 0 {
		b.WriteString("(no commits)\n")
	}
	_ = downDir
	return b.String(), nil
}
