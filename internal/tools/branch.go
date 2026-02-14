package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/amarbel-llc/go-lib-mcp/protocol"
	"github.com/amarbel-llc/go-lib-mcp/server"
	"github.com/friedenberg/grit/internal/git"
)

func registerBranchTools(r *server.ToolRegistry) {
	r.Register(
		"git_branch_list",
		"List branches",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"repo_path": {
					"type": "string",
					"description": "Path to the git repository"
				},
				"remote": {
					"type": "boolean",
					"description": "List remote-tracking branches"
				},
				"all": {
					"type": "boolean",
					"description": "List both local and remote-tracking branches"
				}
			},
			"required": ["repo_path"]
		}`),
		handleGitBranchList,
	)

	r.Register(
		"git_branch_create",
		"Create a new branch",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"repo_path": {
					"type": "string",
					"description": "Path to the git repository"
				},
				"name": {
					"type": "string",
					"description": "Name for the new branch"
				},
				"start_point": {
					"type": "string",
					"description": "Starting point for the new branch (commit, branch, tag)"
				}
			},
			"required": ["repo_path", "name"]
		}`),
		handleGitBranchCreate,
	)

	r.Register(
		"git_checkout",
		"Switch branches or restore working tree files",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"repo_path": {
					"type": "string",
					"description": "Path to the git repository"
				},
				"ref": {
					"type": "string",
					"description": "Branch name or ref to check out"
				},
				"create": {
					"type": "boolean",
					"description": "Create a new branch and check it out (-b)"
				}
			},
			"required": ["repo_path", "ref"]
		}`),
		handleGitCheckout,
	)
}

func handleGitBranchList(ctx context.Context, args json.RawMessage) (*protocol.ToolCallResult, error) {
	var params struct {
		RepoPath string `json:"repo_path"`
		Remote   bool   `json:"remote"`
		All      bool   `json:"all"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return protocol.ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	gitArgs := []string{"branch", "-v"}

	if params.All {
		gitArgs = append(gitArgs, "-a")
	} else if params.Remote {
		gitArgs = append(gitArgs, "-r")
	}

	out, err := git.Run(ctx, params.RepoPath, gitArgs...)
	if err != nil {
		return protocol.ErrorResult(fmt.Sprintf("git branch: %v", err)), nil
	}

	return &protocol.ToolCallResult{
		Content: []protocol.ContentBlock{
			protocol.TextContent(out),
		},
	}, nil
}

func handleGitBranchCreate(ctx context.Context, args json.RawMessage) (*protocol.ToolCallResult, error) {
	var params struct {
		RepoPath   string `json:"repo_path"`
		Name       string `json:"name"`
		StartPoint string `json:"start_point"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return protocol.ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	gitArgs := []string{"branch", params.Name}

	if params.StartPoint != "" {
		gitArgs = append(gitArgs, params.StartPoint)
	}

	out, err := git.Run(ctx, params.RepoPath, gitArgs...)
	if err != nil {
		return protocol.ErrorResult(fmt.Sprintf("git branch create: %v", err)), nil
	}

	if out == "" {
		out = fmt.Sprintf("branch '%s' created", params.Name)
	}

	return &protocol.ToolCallResult{
		Content: []protocol.ContentBlock{
			protocol.TextContent(out),
		},
	}, nil
}

func handleGitCheckout(ctx context.Context, args json.RawMessage) (*protocol.ToolCallResult, error) {
	var params struct {
		RepoPath string `json:"repo_path"`
		Ref      string `json:"ref"`
		Create   bool   `json:"create"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return protocol.ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	gitArgs := []string{"checkout"}

	if params.Create {
		gitArgs = append(gitArgs, "-b")
	}

	gitArgs = append(gitArgs, params.Ref)

	out, err := git.Run(ctx, params.RepoPath, gitArgs...)
	if err != nil {
		return protocol.ErrorResult(fmt.Sprintf("git checkout: %v", err)), nil
	}

	if out == "" {
		out = fmt.Sprintf("switched to branch '%s'", params.Ref)
	}

	return &protocol.ToolCallResult{
		Content: []protocol.ContentBlock{
			protocol.TextContent(out),
		},
	}, nil
}
