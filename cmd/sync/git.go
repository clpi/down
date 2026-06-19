package sync

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	syncGitRoot    string
	syncGitForce   bool
	syncGitVerbose bool
)

func resolveGitRoots(args []string) (workspace, downDir, repoRoot string) {
	workspace = "."
	if len(args) > 0 {
		workspace = args[0]
	}
	if syncGitRoot != "" {
		workspace = syncGitRoot
	}

	downDir = findDownDir(workspace)
	if downDir == "" {
		fmt.Fprintln(os.Stderr, "No .down/ directory found. Run `down init` first.")
		os.Exit(1)
	}

	repoRoot, err := gitRootDir(workspace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Not a git repository: %v\n", err)
		os.Exit(1)
	}
	return workspace, downDir, repoRoot
}

var syncGit = cobra.Command{
	Use:     "git [directory]",
	Aliases: []string{"g"},
	Short:   "Sync git history and diffs into .down/git/",
	Long: `Export git repository history and diffs into markdown under .down/git/.

Creates a compact full-history timeline plus per-commit markdown files with
patches. Incremental sync only writes new commits unless --force is set.

Layout:
  .down/git/
    index.json       sync state (HEAD, branch, known commits)
    history.md       compact full commit timeline
    HEAD.md          current HEAD commit + diff
    working-tree.md  uncommitted changes (if any)
    branches/<name>.md
    commits/<sha>.md per-commit detail + patch`,
	Run: func(cmd *cobra.Command, args []string) {
		_, downDir, repoRoot := resolveGitRoots(args)
		result, err := syncGitHistory(downDir, repoRoot, syncGitForce, syncGitVerbose)
		if err != nil {
			fmt.Fprintf(os.Stderr, "sync git failed: %v\n", err)
			os.Exit(1)
		}
		printGitSyncResult(result)
	},
}

var syncGitStatus = cobra.Command{
	Use:   "status [directory]",
	Short: "Show git repo and .down/git/ sync status",
	Run: func(cmd *cobra.Command, args []string) {
		_, downDir, repoRoot := resolveGitRoots(args)
		st, err := gitStatusReport(downDir, repoRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "sync git status failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(st)
	},
}

var syncGitLog = cobra.Command{
	Use:   "log [directory]",
	Short: "Show recent commits from the synced .down/git/ index",
	Run: func(cmd *cobra.Command, args []string) {
		_, downDir, repoRoot := resolveGitRoots(args)
		out, err := gitLogOutput(downDir, repoRoot, 20)
		if err != nil {
			fmt.Fprintf(os.Stderr, "sync git log failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(out)
	},
}

var syncGitDiff = cobra.Command{
	Use:   "diff [directory]",
	Short: "Show working tree diff (also written to .down/git/working-tree.md on sync)",
	Run: func(cmd *cobra.Command, args []string) {
		_, _, repoRoot := resolveGitRoots(args)
		out, err := gitWorkingDiff(repoRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "sync git diff failed: %v\n", err)
			os.Exit(1)
		}
		if out == "" {
			fmt.Println("No uncommitted changes.")
			return
		}
		fmt.Print(out)
	},
}

func initGit() {
	syncGit.PersistentFlags().StringVar(&syncGitRoot, "root", "", "Workspace root (default: nearest .down/ ancestor)")
	syncGit.PersistentFlags().BoolVarP(&syncGitForce, "force", "f", false, "Regenerate all commit markdown files")
	syncGit.PersistentFlags().BoolVarP(&syncGitVerbose, "verbose", "v", false, "Verbose output")

	syncGit.AddCommand(&syncGitStatus)
	syncGit.AddCommand(&syncGitLog)
	syncGit.AddCommand(&syncGitDiff)

	Sync.AddCommand(&syncGit)
}

func printGitSyncResult(r *GitSyncResult) {
	fmt.Printf("git/: synced %d commit(s)", r.Written)
	if r.Skipped > 0 {
		fmt.Printf(" (%d unchanged)", r.Skipped)
	}
	fmt.Printf(" → %s\n", r.GitDir)
	if r.Branch != "" {
		fmt.Printf("  branch: %s  HEAD: %s\n", r.Branch, shortSHA(r.HEAD))
	}
	if r.WorkingTree {
		fmt.Println("  working-tree.md updated (uncommitted changes)")
	}
}
