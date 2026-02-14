package tools

import "github.com/amarbel-llc/go-lib-mcp/server"

func RegisterAll() *server.ToolRegistry {
	r := server.NewToolRegistry()

	registerStatusTools(r)
	registerLogTools(r)
	registerStagingTools(r)
	registerCommitTools(r)
	registerBranchTools(r)
	registerRemoteTools(r)

	return r
}
