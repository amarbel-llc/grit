package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/amarbel-llc/go-lib-mcp/server"
	"github.com/amarbel-llc/go-lib-mcp/transport"
	"github.com/amarbel-llc/purse-first/purse"
	"github.com/friedenberg/grit/internal/tools"
	intTransport "github.com/friedenberg/grit/internal/transport"
)

func main() {
	sseMode := flag.Bool("sse", false, "Use HTTP/SSE transport instead of stdio")
	port := flag.Int("port", 8080, "Port for HTTP/SSE transport")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "grit — an MCP server exposing git operations\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  grit [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Starts an MCP server on stdio (default) or HTTP/SSE.\n")
		fmt.Fprintf(os.Stderr, "Intended to be launched by an MCP client such as Claude Code.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  grit                     # stdio transport\n")
		fmt.Fprintf(os.Stderr, "  grit --sse --port 8080   # HTTP/SSE transport\n")
	}

	flag.Parse()

	if flag.NArg() == 2 && flag.Arg(0) == "generate-plugin" {
		reason := "Use the grit MCP tool instead of shelling out. When the command uses git -C <path>, pass that path as the repo_path parameter"

		b := purse.NewPluginBuilder("grit").
			Command("grit").
			StdioTransport().
			Mapping("Bash").
			CommandPrefixes("git status").
			Tool("status", "checking repository status").
			Reason(reason).
			Done().
			Mapping("Bash").
			CommandPrefixes("git diff").
			Tool("diff", "viewing changes").
			Reason(reason).
			Done().
			Mapping("Bash").
			CommandPrefixes("git log").
			Tool("log", "viewing commit history").
			Reason(reason).
			Done().
			Mapping("Bash").
			CommandPrefixes("git show").
			Tool("show", "inspecting commits or objects").
			Reason(reason).
			Done().
			Mapping("Bash").
			CommandPrefixes("git blame").
			Tool("blame", "viewing line-by-line authorship").
			Reason(reason).
			Done().
			Mapping("Bash").
			CommandPrefixes("git add").
			Tool("add", "staging files for commit").
			Reason(reason).
			Done().
			Mapping("Bash").
			CommandPrefixes("git reset").
			Tool("reset", "unstaging files").
			Reason(reason).
			Done().
			Mapping("Bash").
			CommandPrefixes("git commit").
			Tool("commit", "creating a new commit").
			Reason(reason).
			Done().
			Mapping("Bash").
			CommandPrefixes("git branch").
			Tool("branch_list", "listing branches").
			Tool("branch_create", "creating a new branch").
			Reason(reason).
			Done().
			Mapping("Bash").
			CommandPrefixes("git checkout", "git switch").
			Tool("checkout", "switching branches").
			Reason(reason).
			Done().
			Mapping("Bash").
			CommandPrefixes("git fetch").
			Tool("fetch", "fetching from a remote").
			Reason(reason).
			Done().
			Mapping("Bash").
			CommandPrefixes("git pull").
			Tool("pull", "pulling changes from a remote").
			Reason(reason).
			Done().
			Mapping("Bash").
			CommandPrefixes("git push").
			Tool("push", "pushing commits to a remote").
			Reason(reason).
			Done().
			Mapping("Bash").
			CommandPrefixes("git remote").
			Tool("remote_list", "listing remotes").
			Reason(reason).
			Done().
			Mapping("Bash").
			CommandPrefixes("git rev-parse").
			Tool("git_rev_parse", "resolving a git revision to its full SHA").
			Reason(reason).
			Done().
			Mapping("Bash").
			CommandPrefixes("git ", "git -C ").
			Tool("status", "checking repository status").
			Tool("diff", "viewing changes").
			Tool("log", "viewing commit history").
			Tool("show", "inspecting commits or objects").
			Tool("blame", "viewing line-by-line authorship").
			Tool("add", "staging files for commit").
			Tool("reset", "unstaging files").
			Tool("commit", "creating a new commit").
			Tool("branch_list", "listing branches").
			Tool("branch_create", "creating a new branch").
			Tool("checkout", "switching branches").
			Tool("fetch", "fetching from a remote").
			Tool("pull", "pulling changes from a remote").
			Tool("push", "pushing commits to a remote").
			Tool("remote_list", "listing remotes").
			Tool("git_rev_parse", "resolving a git revision to its full SHA").
			Reason("Use grit MCP tools for git operations instead of shelling out. When the command uses git -C <path>, pass that path as the repo_path parameter").
			Done()

		p := b.Build()
		dir := flag.Arg(1)

		if err := purse.WritePlugin(dir, p); err != nil {
			log.Fatalf("generating plugin: %v", err)
		}

		if mf := b.BuildMappings(); mf != nil {
			if err := purse.WriteMappings(dir, p.Name, mf); err != nil {
				log.Fatalf("generating mappings: %v", err)
			}
		}

		return
	}

	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "grit: unexpected arguments: %v\n", flag.Args())
		flag.Usage()
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var t transport.Transport

	if *sseMode {
		sse := intTransport.NewSSE(fmt.Sprintf(":%d", *port))
		if err := sse.Start(ctx); err != nil {
			log.Fatalf("starting SSE transport: %v", err)
		}
		defer sse.Close()
		log.Printf("SSE transport listening on %s", sse.Addr())
		t = sse
	} else {
		t = transport.NewStdio(os.Stdin, os.Stdout)
	}

	srv, err := server.New(t, server.Options{
		ServerName:    "grit",
		ServerVersion: "0.1.0",
		Tools:         tools.RegisterAll(),
	})
	if err != nil {
		log.Fatalf("creating server: %v", err)
	}

	if err := srv.Run(ctx); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
