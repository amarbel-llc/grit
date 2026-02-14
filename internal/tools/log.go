package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/amarbel-llc/go-lib-mcp/protocol"
	"github.com/amarbel-llc/go-lib-mcp/server"
	"github.com/friedenberg/grit/internal/git"
)

func registerLogTools(r *server.ToolRegistry) {
	r.Register(
		"git_log",
		"Show commit history",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"repo_path": {
					"type": "string",
					"description": "Path to the git repository"
				},
				"max_count": {
					"type": "integer",
					"description": "Maximum number of commits to show (default 10)"
				},
				"ref": {
					"type": "string",
					"description": "Starting ref (commit, branch, tag)"
				},
				"paths": {
					"type": "array",
					"items": {"type": "string"},
					"description": "Limit to commits affecting these paths"
				},
				"oneline": {
					"type": "boolean",
					"description": "Use condensed one-line format"
				},
				"format": {
					"type": "string",
					"description": "Custom --format string (overrides oneline)"
				},
				"all": {
					"type": "boolean",
					"description": "Show commits from all branches"
				}
			},
			"required": ["repo_path"]
		}`),
		handleGitLog,
	)

	r.Register(
		"git_show",
		"Show a commit, tag, or other git object",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"repo_path": {
					"type": "string",
					"description": "Path to the git repository"
				},
				"ref": {
					"type": "string",
					"description": "Ref to show (commit hash, tag, branch, etc.)"
				}
			},
			"required": ["repo_path", "ref"]
		}`),
		handleGitShow,
	)

	r.Register(
		"git_blame",
		"Show line-by-line authorship of a file",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"repo_path": {
					"type": "string",
					"description": "Path to the git repository"
				},
				"path": {
					"type": "string",
					"description": "File path to blame (relative to repo root)"
				},
				"ref": {
					"type": "string",
					"description": "Blame at a specific ref"
				},
				"line_range": {
					"type": "string",
					"description": "Line range in format START,END (e.g. '10,20')"
				}
			},
			"required": ["repo_path", "path"]
		}`),
		handleGitBlame,
	)
}

func handleGitLog(ctx context.Context, args json.RawMessage) (*protocol.ToolCallResult, error) {
	var params struct {
		RepoPath string   `json:"repo_path"`
		MaxCount int      `json:"max_count"`
		Ref      string   `json:"ref"`
		Paths    []string `json:"paths"`
		Oneline  bool     `json:"oneline"`
		Format   string   `json:"format"`
		All      bool     `json:"all"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return protocol.ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	gitArgs := []string{"log"}

	maxCount := params.MaxCount
	if maxCount <= 0 {
		maxCount = 10
	}
	gitArgs = append(gitArgs, fmt.Sprintf("--max-count=%d", maxCount))

	if params.Format != "" {
		gitArgs = append(gitArgs, fmt.Sprintf("--format=%s", params.Format))
	} else if params.Oneline {
		gitArgs = append(gitArgs, "--oneline")
	}

	if params.All {
		gitArgs = append(gitArgs, "--all")
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
		return protocol.ErrorResult(fmt.Sprintf("git log: %v", err)), nil
	}

	return &protocol.ToolCallResult{
		Content: []protocol.ContentBlock{
			protocol.TextContent(out),
		},
	}, nil
}

func handleGitShow(ctx context.Context, args json.RawMessage) (*protocol.ToolCallResult, error) {
	var params struct {
		RepoPath string `json:"repo_path"`
		Ref      string `json:"ref"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return protocol.ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	out, err := git.Run(ctx, params.RepoPath, "show", params.Ref)
	if err != nil {
		return protocol.ErrorResult(fmt.Sprintf("git show: %v", err)), nil
	}

	return &protocol.ToolCallResult{
		Content: []protocol.ContentBlock{
			protocol.TextContent(out),
		},
	}, nil
}

func handleGitBlame(ctx context.Context, args json.RawMessage) (*protocol.ToolCallResult, error) {
	var params struct {
		RepoPath  string `json:"repo_path"`
		Path      string `json:"path"`
		Ref       string `json:"ref"`
		LineRange string `json:"line_range"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return protocol.ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	gitArgs := []string{"blame"}

	if params.LineRange != "" {
		gitArgs = append(gitArgs, fmt.Sprintf("-L%s", params.LineRange))
	}

	if params.Ref != "" {
		gitArgs = append(gitArgs, params.Ref)
	}

	gitArgs = append(gitArgs, "--", params.Path)

	out, err := git.Run(ctx, params.RepoPath, gitArgs...)
	if err != nil {
		return protocol.ErrorResult(fmt.Sprintf("git blame: %v", err)), nil
	}

	return &protocol.ToolCallResult{
		Content: []protocol.ContentBlock{
			protocol.TextContent(out),
		},
	}, nil
}
