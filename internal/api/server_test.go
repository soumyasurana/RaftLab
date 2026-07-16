package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fakeManager struct {
	health   HealthResponse
	status   StatusResponse
	peers    []PeerResponse
	state    map[string]string
	metrics  MetricsResponse
	snapshot SnapshotResponse

	healthErr   error
	statusErr   error
	peersErr    error
	stateErr    error
	metricsErr  error
	snapshotErr error
	chaosErr    error

	enableCalled  int
	disableCalled int
	resetCalled   int
}

func (f *fakeManager) Health(context.Context) (HealthResponse, error) {
	return f.health, f.healthErr
}

func (f *fakeManager) Status(context.Context) (StatusResponse, error) {
	return f.status, f.statusErr
}

func (f *fakeManager) Peers(context.Context) ([]PeerResponse, error) {
	return f.peers, f.peersErr
}

func (f *fakeManager) StateMachineSnapshot(context.Context) (map[string]string, error) {
	return f.state, f.stateErr
}

func (f *fakeManager) Metrics(context.Context) (MetricsResponse, error) {
	return f.metrics, f.metricsErr
}

func (f *fakeManager) TriggerSnapshot(context.Context) (SnapshotResponse, error) {
	return f.snapshot, f.snapshotErr
}

func (f *fakeManager) EnableChaos(context.Context) error {
	f.enableCalled++
	return f.chaosErr
}

func (f *fakeManager) DisableChaos(context.Context) error {
	f.disableCalled++
	return f.chaosErr
}

func (f *fakeManager) ResetChaos(context.Context) error {
	f.resetCalled++
	return f.chaosErr
}

func TestServerEndpoints(t *testing.T) {
	manager := &fakeManager{
		health: HealthResponse{
			NodeID:      "node-1",
			Role:        "Leader",
			CurrentTerm: 7,
			LeaderID:    "node-1",
			Uptime:      5 * time.Second,
		},
		status: StatusResponse{
			Role:              "Leader",
			CurrentTerm:       7,
			VotedFor:          "node-1",
			CommitIndex:       42,
			LastApplied:       41,
			LastIncludedIndex: 40,
			LastIncludedTerm:  6,
			LogLength:         3,
			Snapshot: SnapshotStatus{
				Available:         true,
				LastIncludedIndex: 40,
				LastIncludedTerm:  6,
			},
		},
		peers: []PeerResponse{{
			PeerID:          "node-2",
			Address:         "127.0.0.1:9002",
			ConnectionState: "connected",
			NextIndex:       43,
			MatchIndex:      42,
		}},
		state: map[string]string{
			"alpha": "bravo",
		},
		metrics: MetricsResponse{
			ElectionsWon:       2,
			ElectionsLost:      1,
			VotesGranted:       5,
			VotesRejected:      3,
			AppendEntriesSent:  11,
			AppendEntriesRecv:  12,
			SnapshotsCreated:   4,
			SnapshotsInstalled: 2,
			RPCFailures:        1,
			LeaderChanges:      3,
			Uptime:             9 * time.Second,
		},
		snapshot: SnapshotResponse{
			SnapshotIndex: 41,
			SnapshotTerm:  7,
			Success:       true,
		},
	}

	server := NewServer(manager, WithBuildVersion("1.2.3"))
	app := server.App()

	tests := []struct {
		name            string
		method          string
		path            string
		wantStatus      int
		wantContentType string
		check           func(*testing.T, *http.Response)
	}{
		{
			name:            "health",
			method:          http.MethodGet,
			path:            "/health",
			wantStatus:      http.StatusOK,
			wantContentType: "application/json",
			check: func(t *testing.T, resp *http.Response) {
				t.Helper()
				var got HealthResponse
				mustDecode(t, resp, &got)
				if got.NodeID != "node-1" || got.Role != "Leader" || got.CurrentTerm != 7 || got.LeaderID != "node-1" || got.BuildVersion != "1.2.3" {
					t.Fatalf("unexpected health response: %+v", got)
				}
				if got.Uptime <= 0 {
					t.Fatalf("expected uptime to be positive, got %s", got.Uptime)
				}
			},
		},
		{
			name:            "status",
			method:          http.MethodGet,
			path:            "/status",
			wantStatus:      http.StatusOK,
			wantContentType: "application/json",
			check: func(t *testing.T, resp *http.Response) {
				t.Helper()
				var got StatusResponse
				mustDecode(t, resp, &got)
				if got.CommitIndex != 42 || got.LogLength != 3 || !got.Snapshot.Available {
					t.Fatalf("unexpected status response: %+v", got)
				}
			},
		},
		{
			name:            "peers",
			method:          http.MethodGet,
			path:            "/peers",
			wantStatus:      http.StatusOK,
			wantContentType: "application/json",
			check: func(t *testing.T, resp *http.Response) {
				t.Helper()
				var got []PeerResponse
				mustDecode(t, resp, &got)
				if len(got) != 1 || got[0].PeerID != "node-2" || got[0].ConnectionState != "connected" {
					t.Fatalf("unexpected peers response: %+v", got)
				}
			},
		},
		{
			name:            "state",
			method:          http.MethodGet,
			path:            "/state",
			wantStatus:      http.StatusOK,
			wantContentType: "application/json",
			check: func(t *testing.T, resp *http.Response) {
				t.Helper()
				var got map[string]string
				mustDecode(t, resp, &got)
				if got["alpha"] != "bravo" {
					t.Fatalf("unexpected state response: %+v", got)
				}
			},
		},
		{
			name:            "metrics",
			method:          http.MethodGet,
			path:            "/metrics",
			wantStatus:      http.StatusOK,
			wantContentType: "application/json",
			check: func(t *testing.T, resp *http.Response) {
				t.Helper()
				var got MetricsResponse
				mustDecode(t, resp, &got)
				if got.ElectionsWon != 2 || got.AppendEntriesSent != 11 || got.LeaderChanges != 3 {
					t.Fatalf("unexpected metrics response: %+v", got)
				}
			},
		},
		{
			name:            "snapshot",
			method:          http.MethodPost,
			path:            "/snapshot",
			wantStatus:      http.StatusOK,
			wantContentType: "application/json",
			check: func(t *testing.T, resp *http.Response) {
				t.Helper()
				var got SnapshotResponse
				mustDecode(t, resp, &got)
				if !got.Success || got.SnapshotIndex != 41 || got.SnapshotTerm != 7 {
					t.Fatalf("unexpected snapshot response: %+v", got)
				}
			},
		},
		{
			name:            "chaos-enable",
			method:          http.MethodPost,
			path:            "/chaos/enable",
			wantStatus:      http.StatusOK,
			wantContentType: "application/json",
			check: func(t *testing.T, resp *http.Response) {
				t.Helper()
				var got ActionResponse
				mustDecode(t, resp, &got)
				if !got.Success {
					t.Fatalf("unexpected chaos enable response: %+v", got)
				}
			},
		},
		{
			name:            "chaos-disable",
			method:          http.MethodPost,
			path:            "/chaos/disable",
			wantStatus:      http.StatusOK,
			wantContentType: "application/json",
			check: func(t *testing.T, resp *http.Response) {
				t.Helper()
				var got ActionResponse
				mustDecode(t, resp, &got)
				if !got.Success {
					t.Fatalf("unexpected chaos disable response: %+v", got)
				}
			},
		},
		{
			name:            "chaos-reset",
			method:          http.MethodPost,
			path:            "/chaos/reset",
			wantStatus:      http.StatusOK,
			wantContentType: "application/json",
			check: func(t *testing.T, resp *http.Response) {
				t.Helper()
				var got ActionResponse
				mustDecode(t, resp, &got)
				if !got.Success {
					t.Fatalf("unexpected chaos reset response: %+v", got)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("unexpected status: got %d want %d", resp.StatusCode, tt.wantStatus)
			}

			if got := resp.Header.Get("Content-Type"); got != tt.wantContentType+"; charset=utf-8" {
				t.Fatalf("unexpected content type: %q", got)
			}

			tt.check(t, resp)
		})
	}

	if manager.enableCalled != 1 || manager.disableCalled != 1 || manager.resetCalled != 1 {
		t.Fatalf("unexpected chaos call counts: enable=%d disable=%d reset=%d", manager.enableCalled, manager.disableCalled, manager.resetCalled)
	}
}

func TestServerErrorHandling(t *testing.T) {
	manager := &fakeManager{
		statusErr: errors.New("boom"),
	}
	server := NewServer(manager)

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	resp, err := server.App().Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}

	var got ErrorResponse
	mustDecode(t, resp, &got)
	if got.Error != "failed to load status" {
		t.Fatalf("unexpected error body: %+v", got)
	}
}

func mustDecode(t *testing.T, resp *http.Response, out any) {
	t.Helper()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if err := json.Unmarshal(body, out); err != nil {
		t.Fatalf("unmarshal body %q: %v", string(body), err)
	}
}
