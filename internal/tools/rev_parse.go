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

func registerRevParseTools(r *server.ToolRegistry) {
	r.Register(
		"git_rev_parse",
		"Resolve a git revision to its full SHA, or resolve special names like HEAD, branch names, tags, and relative refs (e.g. HEAD~3, main^2)",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"repo_path": {
					"type": "string",
					"description": "Path to the git repository"
				},
				"ref": {
					"type": "string",
					"description": "Ref to resolve (e.g. HEAD, main, v1.0, HEAD~3, abc1234)"
				}
			},
			"required": ["repo_path", "ref"]
		}`),
		handleGitRevParse,
	)
}

func handleGitRevParse(ctx context.Context, args json.RawMessage) (*protocol.ToolCallResult, error) {
	var params struct {
		RepoPath string `json:"repo_path"`
		Ref      string `json:"ref"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return protocol.ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	out, err := git.Run(ctx, params.RepoPath, "rev-parse", "--verify", params.Ref)
	if err != nil {
		return protocol.ErrorResult(fmt.Sprintf("git rev-parse: %v", err)), nil
	}

	return jsonResult(git.RevParseResult{
		Resolved: strings.TrimSpace(out),
		Ref:      params.Ref,
	})
}
