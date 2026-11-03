// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/arnakoff/api/internal/adapter/mcp"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/platform/config"
)

// noSources lists no source.
type noSources struct{}

func (noSources) Sources() []app.ListedSource { return nil }

// reloader is a fake reloader that runs until its context is done.
type reloader struct{ stopped chan struct{} }

func (r reloader) Run(ctx context.Context) {
	<-ctx.Done()
	close(r.stopped)
}

func TestServeAnswersUntilTheClientLeavesThenStopsReloading(t *testing.T) {
	t.Parallel()

	server, client := sdk.NewInMemoryTransports()
	r := reloader{stopped: make(chan struct{})}
	done := make(chan error, 1)
	go func() { done <- serve(t.Context(), r, mcp.NewServer(mcp.Options{Sources: noSources{}}), server) }()

	cs, err := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "v0.0.0"}, nil).Connect(t.Context(), client, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := cs.CallTool(t.Context(), &sdk.CallToolParams{Name: "list_sources", Arguments: map[string]any{}})
	if err != nil || res.IsError {
		t.Fatalf("list_sources = %+v, %v; want an answer", res, err)
	}
	if err := cs.Close(); err != nil {
		t.Fatal(err)
	}

	if err := <-done; err != nil {
		t.Errorf("serve() error = %v, want nil once the client leaves", err)
	}
	<-r.stopped
}

func TestRunRefusesAnUnwritableSnapshotDir(t *testing.T) {
	t.Parallel()
	if err := run(t.Context(), config.Config{DataDir: "/dev/null/arnakoff"}); err == nil {
		t.Error("run() error = nil, want the snapshot store error")
	}
}
