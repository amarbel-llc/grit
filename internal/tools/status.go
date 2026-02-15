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
				},
				"context_lines": {
					"type": "integer",
					"description": "Number of context lines around each change (git --unified=N, default 3)"
				},
				"max_patch_lines": {
					"type": "integer",
					"description": "Maximum number of patch output lines. Output is truncated with a truncated flag when exceeded."
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

	result := git.ParseStatus(out)

	return jsonResult(result)
}

func handleGitDiff(ctx context.Context, args json.RawMessage) (*protocol.ToolCallResult, error) {
	var params struct {
		RepoPath      string   `json:"repo_path"`
		Staged        bool     `json:"staged"`
		Ref           string   `json:"ref"`
		Paths         []string `json:"paths"`
		StatOnly      bool     `json:"stat_only"`
		ContextLines  *int     `json:"context_lines"`
		MaxPatchLines int      `json:"max_patch_lines"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return protocol.ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	numstatArgs := []string{"diff", "--numstat"}
	if params.Staged {
		numstatArgs = append(numstatArgs, "--cached")
	}
	if params.Ref != "" {
		numstatArgs = append(numstatArgs, params.Ref)
	}
	if len(params.Paths) > 0 {
		numstatArgs = append(numstatArgs, "--")
		numstatArgs = append(numstatArgs, params.Paths...)
	}

	numstatOut, err := git.Run(ctx, params.RepoPath, numstatArgs...)
	if err != nil {
		return protocol.ErrorResult(fmt.Sprintf("git diff: %v", err)), nil
	}

	stats := git.ParseDiffNumstat(numstatOut)

	var summary git.DiffSummary
	summary.TotalFiles = len(stats)
	for _, s := range stats {
		summary.TotalAdditions += s.Additions
		summary.TotalDeletions += s.Deletions
	}

	result := git.DiffResult{
		Stats:   stats,
		Summary: summary,
	}

	if !params.StatOnly {
		patchArgs := []string{"diff"}
		if params.ContextLines != nil {
			patchArgs = append(patchArgs, fmt.Sprintf("--unified=%d", *params.ContextLines))
		}
		if params.Staged {
			patchArgs = append(patchArgs, "--cached")
		}
		if params.Ref != "" {
			patchArgs = append(patchArgs, params.Ref)
		}
		if len(params.Paths) > 0 {
			patchArgs = append(patchArgs, "--")
			patchArgs = append(patchArgs, params.Paths...)
		}

		patchOut, err := git.Run(ctx, params.RepoPath, patchArgs...)
		if err != nil {
			return protocol.ErrorResult(fmt.Sprintf("git diff: %v", err)), nil
		}

		patch, truncated, truncatedAt := git.TruncatePatch(patchOut, params.MaxPatchLines)
		result.Patch = patch
		result.Truncated = truncated
		result.TruncatedAtLine = truncatedAt
	}

	return jsonResult(result)
}
