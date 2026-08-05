package session_test

import (
	"context"
	"testing"

	"zuri-daemon/pkg/session"
)

func TestSessionManager_Lifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := session.NewManager(tmpDir)
	if err != nil {
		t.Fatalf("failed creating session manager: %v", err)
	}

	ctx := context.Background()

	// 1. Create Session
	sess, err := mgr.CreateSession(ctx, "test-sess-1", "ws-core", "Refactor Auth Service")
	if err != nil {
		t.Fatalf("failed creating session: %v", err)
	}

	if sess.State != session.StateIdle {
		t.Errorf("expected initial state 'idle', got '%s'", sess.State)
	}

	// 2. Record Turn
	if err := mgr.RecordTurn(ctx, sess.ID, "How to handle JWT rotation?", "Use RS256 key pairs with 5m cache TTL."); err != nil {
		t.Fatalf("failed recording turn: %v", err)
	}

	// 3. Update State
	if err := mgr.UpdateState(ctx, sess.ID, session.StateExecuting, "Executing JWT key rotation fix"); err != nil {
		t.Fatalf("failed updating state: %v", err)
	}

	// 4. Retrieve & Verify Persistence
	retrieved, err := mgr.GetSession(ctx, sess.ID)
	if err != nil {
		t.Fatalf("failed retrieving session: %v", err)
	}

	if retrieved.State != session.StateExecuting {
		t.Errorf("expected state 'executing', got '%s'", retrieved.State)
	}

	if len(retrieved.QueryHistory) != 1 {
		t.Fatalf("expected 1 query turn, got %d", len(retrieved.QueryHistory))
	}
}
