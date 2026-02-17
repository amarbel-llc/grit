package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/amarbel-llc/purse-first/libs/go-mcp/command"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/protocol"
	"github.com/friedenberg/grit/internal/git"
)

func registerLogCommands(app *command.App) {
	app.AddCommand(&command.Command{
		Name:        "log",
		Description: "Show commit history as structured JSON",
		Params: []command.Param{
			{Name: "repo_path", Type: command.String, Description: "Path to the git repository", Required: true},
			{Name: "max_count", Type: command.Int, Description: "Maximum number of commits to show (default 10)"},
			{Name: "ref", Type: command.String, Description: "Starting ref (commit, branch, tag)"},
			{Name: "paths", Type: command.Array, Description: "Limit to commits affecting these paths"},
			{Name: "all", Type: command.Bool, Description: "Show commits from all branches"},
		},
		MapsBash: []command.BashMapping{
			{Prefixes: []string{"git log"}, UseWhen: "viewing commit history"},
		},
		RunMCP: handleGitLog,
	})

	app.AddCommand(&command.Command{
		Name:        "show",
		Description: "Show a commit, tag, or other git object",
		Params: []command.Param{
			{Name: "repo_path", Type: command.String, Description: "Path to the git repository", Required: true},
			{Name: "ref", Type: command.String, Description: "Ref to show (commit hash, tag, branch, etc.)", Required: true},
			{Name: "context_lines", Type: command.Int, Description: "Number of context lines around each change (git --unified=N, default 3)"},
			{Name: "max_patch_lines", Type: command.Int, Description: "Maximum number of patch output lines. Output is truncated with a truncated flag when exceeded."},
		},
		MapsBash: []command.BashMapping{
			{Prefixes: []string{"git show"}, UseWhen: "inspecting commits or objects"},
		},
		RunMCP: handleGitShow,
	})

	app.AddCommand(&command.Command{
		Name:        "blame",
		Description: "Show line-by-line authorship of a file",
		Params: []command.Param{
			{Name: "repo_path", Type: command.String, Description: "Path to the git repository", Required: true},
			{Name: "path", Type: command.String, Description: "File path to blame (relative to repo root)", Required: true},
			{Name: "ref", Type: command.String, Description: "Blame at a specific ref"},
			{Name: "line_range", Type: command.String, Description: "Line range in format START,END (e.g. '10,20')"},
		},
		MapsBash: []command.BashMapping{
			{Prefixes: []string{"git blame"}, UseWhen: "viewing line-by-line authorship"},
		},
		RunMCP: handleGitBlame,
	})
}

func handleGitLog(ctx context.Context, args json.RawMessage) (*protocol.ToolCallResult, error) {
	var params struct {
		RepoPath string   `json:"repo_path"`
		MaxCount int      `json:"max_count"`
		Ref      string   `json:"ref"`
		Paths    []string `json:"paths"`
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
	gitArgs = append(gitArgs, fmt.Sprintf("--format=%s", git.LogFormat))

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

	entries := git.ParseLog(out)

	return jsonResult(entries)
}

func handleGitShow(ctx context.Context, args json.RawMessage) (*protocol.ToolCallResult, error) {
	var params struct {
		RepoPath      string `json:"repo_path"`
		Ref           string `json:"ref"`
		ContextLines  *int   `json:"context_lines"`
		MaxPatchLines int    `json:"max_patch_lines"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return protocol.ErrorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	metadataOut, err := git.Run(ctx, params.RepoPath, "show", "--no-patch", fmt.Sprintf("--format=%s", git.ShowFormat), params.Ref)
	if err != nil {
		// Fall back to raw output for non-commit objects (tags, blobs)
		out, fallbackErr := git.Run(ctx, params.RepoPath, "show", params.Ref)
		if fallbackErr != nil {
			return protocol.ErrorResult(fmt.Sprintf("git show: %v", err)), nil
		}
		return &protocol.ToolCallResult{
			Content: []protocol.ContentBlock{
				protocol.TextContent(out),
			},
		}, nil
	}

	numstatOut, err := git.Run(ctx, params.RepoPath, "show", "--numstat", "--format=", params.Ref)
	if err != nil {
		numstatOut = ""
	}

	diffArgs := []string{"diff"}
	if params.ContextLines != nil {
		diffArgs = append(diffArgs, fmt.Sprintf("--unified=%d", *params.ContextLines))
	}
	diffArgs = append(diffArgs, params.Ref+"~1", params.Ref)

	patchOut, err := git.Run(ctx, params.RepoPath, diffArgs...)
	if err != nil {
		patchOut = ""
	}

	result := git.ParseShow(metadataOut, numstatOut, patchOut)

	patch, truncated, truncatedAt := git.TruncatePatch(result.Patch, params.MaxPatchLines)
	result.Patch = patch
	result.Truncated = truncated
	result.TruncatedAtLine = truncatedAt

	return jsonResult(result)
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

	gitArgs := []string{"blame", "--porcelain"}

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

	lines := git.ParseBlame(out)

	return jsonResult(lines)
}
