package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/amarbel-llc/purse-first/libs/go-mcp/command"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/protocol"
	"github.com/friedenberg/grit/internal/git"
)

func registerStagingCommands(app *command.App) {
	app.AddCommand(&command.Command{
		Name:        "add",
		Description: command.Description{Short: "Stage files for commit"},
		Params: []command.Param{
			{Name: "repo_path", Type: command.String, Description: "Path to the git repository", Required: true},
			{Name: "paths", Type: command.Array, Description: "File paths to stage (relative to repo root)", Required: true},
		},
		MapsBash: []command.BashMapping{
			{Prefixes: []string{"git add"}, UseWhen: "staging files for commit"},
		},
		RunMCP: handleGitAdd,
	})

	app.AddCommand(&command.Command{
		Name:        "reset",
		Description: command.Description{Short: "Unstage files (soft reset only, does not modify working tree)"},
		Params: []command.Param{
			{Name: "repo_path", Type: command.String, Description: "Path to the git repository", Required: true},
			{Name: "paths", Type: command.Array, Description: "File paths to unstage (relative to repo root)", Required: true},
		},
		MapsBash: []command.BashMapping{
			{Prefixes: []string{"git reset"}, UseWhen: "unstaging files"},
		},
		RunMCP: handleGitReset,
	})
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

	if _, err := git.Run(ctx, params.RepoPath, gitArgs...); err != nil {
		return protocol.ErrorResult(fmt.Sprintf("git add: %v", err)), nil
	}

	return jsonResult(git.MutationResult{
		Status: "staged",
		Paths:  params.Paths,
	})
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

	if _, err := git.Run(ctx, params.RepoPath, gitArgs...); err != nil {
		return protocol.ErrorResult(fmt.Sprintf("git reset: %v", err)), nil
	}

	return jsonResult(git.MutationResult{
		Status: "unstaged",
		Paths:  params.Paths,
	})
}
