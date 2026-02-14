package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/amarbel-llc/go-lib-mcp/protocol"
	"github.com/amarbel-llc/go-lib-mcp/server"
	"github.com/friedenberg/grit/internal/git"
)

func registerStatusTools(r *server.ToolRegistry) {
	r.Register(
		"git_status",
		"Show working tree status with machine-readable output",
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
		handleGitStatus,
	)

	r.Register(
		"git_diff",
		"Show changes in the working tree or between commits",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"repo_path": {
					"type": "string",
					"description": "Path to the git repository"
				},
				"staged": {
					"type": "boolean",
					"description": "Show staged changes (--cached)"
				},
				"ref": {
					"type": "string",
					"description": "Diff against a specific ref (commit, branch, tag)"
				},
				"paths": {
					"type": "array",
					"items": {"type": "string"},
					"description": "Limit diff to specific paths"
				},
				"stat_only": {
					"type": "boolean",
					"description": "Show only diffstat summary"
				}
			},
			"required": ["repo_path"]
		}`),
		handleGitDiff,
	)
}

func handleGitStatus(ctx context.Context, args json.RawMessage) (*protocol.ToolCallResult, error) {
	var params struct {
		RepoPath string `json:"repo_path"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return protocol.ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	out, err := git.Run(ctx, params.RepoPath, "status", "--porcelain=v2", "--branch")
	if err != nil {
		return protocol.ErrorResult(fmt.Sprintf("git status: %v", err)), nil
	}

	return &protocol.ToolCallResult{
		Content: []protocol.ContentBlock{
			protocol.TextContent(out),
		},
	}, nil
}

func handleGitDiff(ctx context.Context, args json.RawMessage) (*protocol.ToolCallResult, error) {
	var params struct {
		RepoPath string   `json:"repo_path"`
		Staged   bool     `json:"staged"`
		Ref      string   `json:"ref"`
		Paths    []string `json:"paths"`
		StatOnly bool     `json:"stat_only"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return protocol.ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	gitArgs := []string{"diff"}

	if params.Staged {
		gitArgs = append(gitArgs, "--cached")
	}

	if params.StatOnly {
		gitArgs = append(gitArgs, "--stat")
	}

	if params.Ref != "" {
		gitArgs = append(gitArgs, params.Ref)
	}

	if len(params.Paths) > 0 {
		gitArgs = append(gitArgs, "--")
		gitArgs = append(gitArgs, params.Paths...)
	}

	out, err := git.Run(ctx, params.RepoPath, gitArgs...)
	if err != nil {
		return protocol.ErrorResult(fmt.Sprintf("git diff: %v", err)), nil
	}

	return &protocol.ToolCallResult{
		Content: []protocol.ContentBlock{
			protocol.TextContent(out),
		},
	}, nil
}
