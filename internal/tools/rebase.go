package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/amarbel-llc/purse-first/libs/go-mcp/command"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/protocol"
	"github.com/friedenberg/grit/internal/git"
)

func registerRebaseCommands(app *command.App) {
	app.AddCommand(&command.Command{
		Name:        "rebase",
		Description: command.Description{Short: "Rebase current branch onto another ref (blocked on main/master for safety)"},
		Params: []command.Param{
			{Name: "repo_path", Type: command.String, Description: "Path to the git repository", Required: true},
			{Name: "upstream", Type: command.String, Description: "Ref to rebase onto (branch, tag, commit)"},
			{Name: "branch", Type: command.String, Description: "Branch to rebase (defaults to current branch)"},
			{Name: "autostash", Type: command.Bool, Description: "Automatically stash/unstash uncommitted changes"},
			{Name: "continue", Type: command.Bool, Description: "Continue rebase after resolving conflicts"},
			{Name: "abort", Type: command.Bool, Description: "Abort current rebase operation"},
			{Name: "skip", Type: command.Bool, Description: "Skip current commit and continue rebase"},
		},
		MapsTools: []command.ToolMapping{
			{Replaces: "Bash", CommandPrefixes: []string{"git rebase"}, UseWhen: "rebasing a branch"},
		},
		RunMCP: handleGitRebase,
	})
}

func handleGitRebase(ctx context.Context, args json.RawMessage) (*protocol.ToolCallResult, error) {
	var params struct {
		RepoPath  string `json:"repo_path"`
		Upstream  string `json:"upstream"`
		Branch    string `json:"branch"`
		Autostash bool   `json:"autostash"`
		Continue  bool   `json:"continue"`
		Abort     bool   `json:"abort"`
		Skip      bool   `json:"skip"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return protocol.ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	// Validate mutually exclusive operations
	opCount := 0
	if params.Continue {
		opCount++
	}
	if params.Abort {
		opCount++
	}
	if params.Skip {
		opCount++
	}
	if params.Upstream != "" {
		opCount++
	}

	if opCount > 1 {
		return protocol.ErrorResult("only one of upstream, continue, abort, or skip can be specified"), nil
	}

	if opCount == 0 {
		return protocol.ErrorResult("must specify upstream (for new rebase) or continue/abort/skip (for existing rebase)"), nil
	}

	// Handle abort
	if params.Abort {
		if _, err := git.Run(ctx, params.RepoPath, "rebase", "--abort"); err != nil {
			return protocol.ErrorResult(fmt.Sprintf("git rebase --abort: %v", err)), nil
		}

		return jsonResult(git.RebaseResult{
			Status: "aborted",
		})
	}

	// Handle continue
	if params.Continue {
		out, err := git.Run(ctx, params.RepoPath, "rebase", "--continue")
		if err != nil {
			// Check if there are still conflicts
			if strings.Contains(err.Error(), "fix conflicts") || strings.Contains(err.Error(), "still have conflicts") {
				conflicts := extractConflictFiles(ctx, params.RepoPath)
				return jsonResult(git.RebaseResult{
					Status:    "conflict",
					Conflicts: conflicts,
				})
			}
			return protocol.ErrorResult(fmt.Sprintf("git rebase --continue: %v", err)), nil
		}

		return jsonResult(git.RebaseResult{
			Status:  "completed",
			Summary: strings.TrimSpace(out),
		})
	}

	// Handle skip
	if params.Skip {
		out, err := git.Run(ctx, params.RepoPath, "rebase", "--skip")
		if err != nil {
			return protocol.ErrorResult(fmt.Sprintf("git rebase --skip: %v", err)), nil
		}

		return jsonResult(git.RebaseResult{
			Status:  "skipped",
			Summary: strings.TrimSpace(out),
		})
	}

	// Handle new rebase
	if params.Upstream != "" {
		// Determine which branch is being rebased
		branchToRebase := params.Branch
		if branchToRebase == "" {
			branchOut, err := git.Run(ctx, params.RepoPath, "rev-parse", "--abbrev-ref", "HEAD")
			if err == nil {
				branchToRebase = strings.TrimSpace(branchOut)
			}
		}

		// Safety: block rebasing main/master
		if branchToRebase == "main" || branchToRebase == "master" {
			return protocol.ErrorResult("rebasing main/master is blocked for safety"), nil
		}

		// Check for existing rebase state
		rebaseDir := ".git/rebase-merge"
		if _, err := git.Run(ctx, params.RepoPath, "test", "-d", rebaseDir); err == nil {
			return protocol.ErrorResult("a rebase operation is already in progress; use continue, abort, or skip"), nil
		}

		gitArgs := []string{"rebase"}

		if params.Autostash {
			gitArgs = append(gitArgs, "--autostash")
		}

		gitArgs = append(gitArgs, params.Upstream)

		if params.Branch != "" {
			gitArgs = append(gitArgs, params.Branch)
		}

		out, err := git.Run(ctx, params.RepoPath, gitArgs...)
		if err != nil {
			// Check for conflicts
			if strings.Contains(err.Error(), "CONFLICT") || strings.Contains(err.Error(), "could not apply") {
				conflicts := extractConflictFiles(ctx, params.RepoPath)
				return jsonResult(git.RebaseResult{
					Status:    "conflict",
					Branch:    branchToRebase,
					Upstream:  params.Upstream,
					Conflicts: conflicts,
				})
			}
			return protocol.ErrorResult(fmt.Sprintf("git rebase: %v", err)), nil
		}

		result := git.RebaseResult{
			Status:   "completed",
			Branch:   branchToRebase,
			Upstream: params.Upstream,
			Summary:  strings.TrimSpace(out),
		}

		if strings.Contains(out, "is up to date") {
			result.Status = "up_to_date"
			result.Summary = ""
		}

		return jsonResult(result)
	}

	return protocol.ErrorResult("unexpected state: no operation specified"), nil
}

func extractConflictFiles(ctx context.Context, repoPath string) []string {
	out, err := git.Run(ctx, repoPath, "diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	conflicts := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			conflicts = append(conflicts, line)
		}
	}

	return conflicts
}
