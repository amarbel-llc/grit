package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/amarbel-llc/go-lib-mcp/protocol"
	"github.com/amarbel-llc/go-lib-mcp/server"
	"github.com/friedenberg/grit/internal/git"
)

func registerStagingTools(r *server.ToolRegistry) {
	r.Register(
		"git_add",
		"Stage files for commit",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"repo_path": {
					"type": "string",
					"description": "Path to the git repository"
				},
				"paths": {
					"type": "array",
					"items": {"type": "string"},
					"description": "File paths to stage (relative to repo root)"
				}
			},
			"required": ["repo_path", "paths"]
		}`),
		handleGitAdd,
	)

	r.Register(
		"git_reset",
		"Unstage files (soft reset only, does not modify working tree)",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"repo_path": {
					"type": "string",
					"description": "Path to the git repository"
				},
				"paths": {
					"type": "array",
					"items": {"type": "string"},
					"description": "File paths to unstage (relative to repo root)"
				}
			},
			"required": ["repo_path", "paths"]
		}`),
		handleGitReset,
	)
}

func handleGitAdd(ctx context.Context, args json.RawMessage) (*protocol.ToolCallResult, error) {
	var params struct {
		RepoPath string   `json:"repo_path"`
		Paths    []string `json:"paths"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return protocol.ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	gitArgs := []string{"add", "--"}
	gitArgs = append(gitArgs, params.Paths...)

	out, err := git.Run(ctx, params.RepoPath, gitArgs...)
	if err != nil {
		return protocol.ErrorResult(fmt.Sprintf("git add: %v", err)), nil
	}

	if out == "" {
		out = "files staged successfully"
	}

	return &protocol.ToolCallResult{
		Content: []protocol.ContentBlock{
			protocol.TextContent(out),
		},
	}, nil
}

func handleGitReset(ctx context.Context, args json.RawMessage) (*protocol.ToolCallResult, error) {
	var params struct {
		RepoPath string   `json:"repo_path"`
		Paths    []string `json:"paths"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return protocol.ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	gitArgs := []string{"reset", "HEAD", "--"}
	gitArgs = append(gitArgs, params.Paths...)

	out, err := git.Run(ctx, params.RepoPath, gitArgs...)
	if err != nil {
		return protocol.ErrorResult(fmt.Sprintf("git reset: %v", err)), nil
	}

	if out == "" {
		out = "files unstaged successfully"
	}

	return &protocol.ToolCallResult{
		Content: []protocol.ContentBlock{
			protocol.TextContent(out),
		},
	}, nil
}
