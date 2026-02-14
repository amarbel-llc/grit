package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/amarbel-llc/go-lib-mcp/protocol"
	"github.com/amarbel-llc/go-lib-mcp/server"
	"github.com/friedenberg/grit/internal/git"
)

func registerRemoteTools(r *server.ToolRegistry) {
	r.Register(
		"git_fetch",
		"Fetch from a remote repository",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"repo_path": {
					"type": "string",
					"description": "Path to the git repository"
				},
				"remote": {
					"type": "string",
					"description": "Remote name (default origin)"
				},
				"prune": {
					"type": "boolean",
					"description": "Prune remote-tracking branches no longer on remote"
				},
				"all": {
					"type": "boolean",
					"description": "Fetch from all remotes"
				}
			},
			"required": ["repo_path"]
		}`),
		handleGitFetch,
	)

	r.Register(
		"git_pull",
		"Pull changes from a remote repository",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"repo_path": {
					"type": "string",
					"description": "Path to the git repository"
				},
				"remote": {
					"type": "string",
					"description": "Remote name (default origin)"
				},
				"branch": {
					"type": "string",
					"description": "Remote branch to pull"
				},
				"rebase": {
					"type": "boolean",
					"description": "Rebase instead of merge"
				}
			},
			"required": ["repo_path"]
		}`),
		handleGitPull,
	)

	r.Register(
		"git_push",
		"Push commits to a remote repository (force push blocked on main/master)",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"repo_path": {
					"type": "string",
					"description": "Path to the git repository"
				},
				"remote": {
					"type": "string",
					"description": "Remote name (default origin)"
				},
				"branch": {
					"type": "string",
					"description": "Branch to push"
				},
				"set_upstream": {
					"type": "boolean",
					"description": "Set upstream tracking reference (-u)"
				},
				"force": {
					"type": "boolean",
					"description": "Force push (blocked on main/master branches)"
				}
			},
			"required": ["repo_path"]
		}`),
		handleGitPush,
	)

	r.Register(
		"git_remote_list",
		"List remotes with their URLs",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"repo_path": {
					"type": "string",
					"description": "Path to the git repository"
				}
			},
			"required": ["repo_path"]
		}`),
		handleGitRemoteList,
	)
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

	out, err := git.Run(ctx, params.RepoPath, gitArgs...)
	if err != nil {
		return protocol.ErrorResult(fmt.Sprintf("git fetch: %v", err)), nil
	}

	if out == "" {
		out = "fetch completed"
	}

	return &protocol.ToolCallResult{
		Content: []protocol.ContentBlock{
			protocol.TextContent(out),
		},
	}, nil
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

	return &protocol.ToolCallResult{
		Content: []protocol.ContentBlock{
			protocol.TextContent(out),
		},
	}, nil
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

	out, err := git.Run(ctx, params.RepoPath, gitArgs...)
	if err != nil {
		return protocol.ErrorResult(fmt.Sprintf("git push: %v", err)), nil
	}

	if out == "" {
		out = "push completed"
	}

	return &protocol.ToolCallResult{
		Content: []protocol.ContentBlock{
			protocol.TextContent(out),
		},
	}, nil
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

	if out == "" {
		out = "no remotes configured"
	}

	return &protocol.ToolCallResult{
		Content: []protocol.ContentBlock{
			protocol.TextContent(out),
		},
	}, nil
}
