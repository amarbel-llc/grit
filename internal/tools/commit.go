package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/amarbel-llc/go-lib-mcp/protocol"
	"github.com/amarbel-llc/go-lib-mcp/server"
	"github.com/friedenberg/grit/internal/git"
)

func registerCommitTools(r *server.ToolRegistry) {
	r.Register(
		"git_commit",
		"Create a new commit with staged changes",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"repo_path": {
					"type": "string",
					"description": "Path to the git repository"
				},
				"message": {
					"type": "string",
					"description": "Commit message"
				}
			},
			"required": ["repo_path", "message"]
		}`),
		handleGitCommit,
	)
}

func handleGitCommit(ctx context.Context, args json.RawMessage) (*protocol.ToolCallResult, error) {
	var params struct {
		RepoPath string `json:"repo_path"`
		Message  string `json:"message"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return protocol.ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	out, err := git.Run(ctx, params.RepoPath, "commit", "-m", params.Message)
	if err != nil {
		return protocol.ErrorResult(fmt.Sprintf("git commit: %v", err)), nil
	}

	result := git.ParseCommit(out)

	return jsonResult(result)
}
