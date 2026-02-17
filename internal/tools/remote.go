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

func registerRemoteCommands(app *command.App) {
	app.AddCommand(&command.Command{
		Name:        "fetch",
		Description: "Fetch from a remote repository",
		Params: []command.Param{
			{Name: "repo_path", Type: command.String, Description: "Path to the git repository", Required: true},
			{Name: "remote", Type: command.String, Description: "Remote name (default origin)"},
			{Name: "prune", Type: command.Bool, Description: "Prune remote-tracking branches no longer on remote"},
			{Name: "all", Type: command.Bool, Description: "Fetch from all remotes"},
		},
		MapsBash: []command.BashMapping{
			{Prefixes: []string{"git fetch"}, UseWhen: "fetching from a remote"},
		},
		RunMCP: handleGitFetch,
	})

	app.AddCommand(&command.Command{
		Name:        "pull",
		Description: "Pull changes from a remote repository",
		Params: []command.Param{
			{Name: "repo_path", Type: command.String, Description: "Path to the git repository", Required: true},
			{Name: "remote", Type: command.String, Description: "Remote name (default origin)"},
			{Name: "branch", Type: command.String, Description: "Remote branch to pull"},
			{Name: "rebase", Type: command.Bool, Description: "Rebase instead of merge"},
		},
		MapsBash: []command.BashMapping{
			{Prefixes: []string{"git pull"}, UseWhen: "pulling changes from a remote"},
		},
		RunMCP: handleGitPull,
	})

	app.AddCommand(&command.Command{
		Name:        "push",
		Description: "Push commits to a remote repository (force push blocked on main/master)",
		Params: []command.Param{
			{Name: "repo_path", Type: command.String, Description: "Path to the git repository", Required: true},
			{Name: "remote", Type: command.String, Description: "Remote name (default origin)"},
			{Name: "branch", Type: command.String, Description: "Branch to push"},
			{Name: "set_upstream", Type: command.Bool, Description: "Set upstream tracking reference (-u)"},
			{Name: "force", Type: command.Bool, Description: "Force push (blocked on main/master branches)"},
		},
		MapsBash: []command.BashMapping{
			{Prefixes: []string{"git push"}, UseWhen: "pushing commits to a remote"},
		},
		RunMCP: handleGitPush,
	})

	app.AddCommand(&command.Command{
		Name:        "remote_list",
		Description: "List remotes with their URLs",
		Params: []command.Param{
			{Name: "repo_path", Type: command.String, Description: "Path to the git repository", Required: true},
		},
		MapsBash: []command.BashMapping{
			{Prefixes: []string{"git remote"}, UseWhen: "listing remotes"},
		},
		RunMCP: handleGitRemoteList,
	})
}

func handleGitFetch(ctx context.Context, args json.RawMessage) (*protocol.ToolCallResult, error) {
	var params struct {
		RepoPath string `json:"repo_path"`
		Remote   string `json:"remote"`
		Prune    bool   `json:"prune"`
		All      bool   `json:"all"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return protocol.ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	gitArgs := []string{"fetch"}

	if params.Prune {
		gitArgs = append(gitArgs, "--prune")
	}

	if params.All {
		gitArgs = append(gitArgs, "--all")
	} else if params.Remote != "" {
		gitArgs = append(gitArgs, params.Remote)
	}

	if _, err := git.Run(ctx, params.RepoPath, gitArgs...); err != nil {
		return protocol.ErrorResult(fmt.Sprintf("git fetch: %v", err)), nil
	}

	remote := params.Remote
	if remote == "" && !params.All {
		remote = "origin"
	}

	return jsonResult(git.MutationResult{
		Status: "fetched",
		Remote: remote,
		All:    params.All,
		Prune:  params.Prune,
	})
}

func handleGitPull(ctx context.Context, args json.RawMessage) (*protocol.ToolCallResult, error) {
	var params struct {
		RepoPath string `json:"repo_path"`
		Remote   string `json:"remote"`
		Branch   string `json:"branch"`
		Rebase   bool   `json:"rebase"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return protocol.ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	gitArgs := []string{"pull"}

	if params.Rebase {
		gitArgs = append(gitArgs, "--rebase")
	}

	if params.Remote != "" {
		gitArgs = append(gitArgs, params.Remote)
	}

	if params.Branch != "" {
		gitArgs = append(gitArgs, params.Branch)
	}

	out, err := git.Run(ctx, params.RepoPath, gitArgs...)
	if err != nil {
		return protocol.ErrorResult(fmt.Sprintf("git pull: %v", err)), nil
	}

	result := git.PullResult{
		Status:  "pulled",
		Summary: strings.TrimSpace(out),
	}

	if strings.Contains(out, "Already up to date") {
		result.Status = "already_up_to_date"
		result.Summary = ""
	}

	return jsonResult(result)
}

func handleGitPush(ctx context.Context, args json.RawMessage) (*protocol.ToolCallResult, error) {
	var params struct {
		RepoPath    string `json:"repo_path"`
		Remote      string `json:"remote"`
		Branch      string `json:"branch"`
		SetUpstream bool   `json:"set_upstream"`
		Force       bool   `json:"force"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return protocol.ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	if params.Force {
		branch := params.Branch
		if branch == "" {
			// Determine current branch to check protection
			branchOut, err := git.Run(ctx, params.RepoPath, "rev-parse", "--abbrev-ref", "HEAD")
			if err == nil {
				branch = strings.TrimSpace(branchOut)
			}
		}

		if branch == "main" || branch == "master" {
			return protocol.ErrorResult("force push to main/master is blocked for safety"), nil
		}
	}

	gitArgs := []string{"push"}

	if params.Force {
		gitArgs = append(gitArgs, "--force")
	}

	if params.SetUpstream {
		gitArgs = append(gitArgs, "-u")
	}

	if params.Remote != "" {
		gitArgs = append(gitArgs, params.Remote)
	}

	if params.Branch != "" {
		gitArgs = append(gitArgs, params.Branch)
	}

	if _, err := git.Run(ctx, params.RepoPath, gitArgs...); err != nil {
		return protocol.ErrorResult(fmt.Sprintf("git push: %v", err)), nil
	}

	return jsonResult(git.MutationResult{
		Status:      "pushed",
		Remote:      params.Remote,
		Branch:      params.Branch,
		SetUpstream: params.SetUpstream,
		Force:       params.Force,
	})
}

func handleGitRemoteList(ctx context.Context, args json.RawMessage) (*protocol.ToolCallResult, error) {
	var params struct {
		RepoPath string `json:"repo_path"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return protocol.ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	out, err := git.Run(ctx, params.RepoPath, "remote", "-v")
	if err != nil {
		return protocol.ErrorResult(fmt.Sprintf("git remote: %v", err)), nil
	}

	remotes := git.ParseRemoteList(out)

	return jsonResult(remotes)
}
