package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/amarbel-llc/purse-first/libs/go-mcp/command"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/protocol"
	"github.com/friedenberg/grit/internal/git"
)

func registerRevParseCommands(app *command.App) {
	app.AddCommand(&command.Command{
		Name:        "git_rev_parse",
		Description: "Resolve a git revision to its full SHA, or resolve special names like HEAD, branch names, tags, and relative refs (e.g. HEAD~3, main^2)",
		Params: []command.Param{
			{Name: "repo_path", Type: command.String, Description: "Path to the git repository", Required: true},
			{Name: "ref", Type: command.String, Description: "Ref to resolve (e.g. HEAD, main, v1.0, HEAD~3, abc1234)", Required: true},
		},
		MapsBash: []command.BashMapping{
			{Prefixes: []string{"git rev-parse"}, UseWhen: "resolving a git revision to its full SHA"},
		},
		RunMCP: handleGitRevParse,
	})
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
