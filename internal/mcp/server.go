package mcp

import (
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/adriangitvitz/kurai/internal/server"
)

type Options struct {
	KeepAlive time.Duration
}

func NewToolServer(svc *server.Server, version string, opts *Options) *mcpsdk.Server {
	var keepAlive time.Duration
	if opts != nil {
		keepAlive = opts.KeepAlive
	}

	srv := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "kurai",
		Version: version,
	}, &mcpsdk.ServerOptions{
		Instructions: instructions,
		KeepAlive:    keepAlive,
	})

	registerSearchTools(srv, svc)
	registerExtractTool(srv, svc)
	registerFindSimilarTool(srv, svc)
	return srv
}
