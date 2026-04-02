package server

import (
	"net/http/httptest"
	"testing"

	"github.com/46labs/permitio/pkg/config"
	"github.com/46labs/permitio/pkg/store"
	permitConfig "github.com/permitio/permit-golang/pkg/config"
	"github.com/permitio/permit-golang/pkg/permit"
)

type testEnv struct {
	server *Server
	ts     *httptest.Server
	client *permit.Client
	store  *store.Store
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	cfg := &config.Config{Port: 0}
	st := store.New()
	srv := NewWithStore(cfg, st)
	ts := httptest.NewServer(srv.Handler())

	client := permit.NewPermit(
		permitConfig.NewConfigBuilder("test-api-key").
			WithPdpUrl(ts.URL).
			WithApiUrl(ts.URL).
			Build(),
	)

	t.Cleanup(func() { ts.Close() })

	return &testEnv{
		server: srv,
		ts:     ts,
		client: client,
		store:  st,
	}
}
