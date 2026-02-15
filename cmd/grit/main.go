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
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "grit — an MCP server exposing git operations\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  grit\n\n")
		fmt.Fprintf(os.Stderr, "Starts an MCP server on stdio (JSON-RPC over stdin/stdout).\n")
		fmt.Fprintf(os.Stderr, "Intended to be launched by an MCP client such as Claude Code.\n\n")
		fmt.Fprintf(os.Stderr, "Example:\n")
		fmt.Fprintf(os.Stderr, "  claude mcp add grit -- grit\n")
	}

	flag.Parse()

	if flag.NArg() == 2 && flag.Arg(0) == "generate-plugin" {
		b := purse.NewPluginBuilder("grit").
			Command("grit").
			StdioTransport().
			Mapping("Bash").
			CommandPrefixes("git ").
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
			Reason("Use grit MCP tools for git operations instead of shelling out").
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

	t := transport.NewStdio(os.Stdin, os.Stdout)

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
