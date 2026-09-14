package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kosc/chessweb/internal/chess"
)

func createTestGame(t *testing.T, h http.Handler) gameStateResponse {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/games", strings.NewReader(`{"botMaxPly":2,"botMoveTimeMs":5000}`)))
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var state gameStateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		store.mu.Lock()
		delete(store.games, state.ID)
		store.mu.Unlock()
	})
	return state
}

func TestMoveRejectsBotTurn(t *testing.T) {
	h := NewHTTPHandler()
	state := createTestGame(t, h)
	g := storeGet(state.ID)
	g.mu.Lock()
	g.fen = strings.Replace(state.FEN, " w ", " b ", 1)
	g.sideToMove = "black"
	g.clockEnabled = true
	g.whiteSec, g.blackSec = 600, 600
	g.turnStartedAt = time.Now().Add(-10 * time.Second)
	before := toGameStateResponse(g)
	started := g.turnStartedAt
	g.mu.Unlock()

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/games/"+state.ID+"/move", strings.NewReader(`{"move":"e7e5"}`)))
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "not your turn") {
		t.Fatalf("expected turn conflict, got %d %s", w.Code, w.Body.String())
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	if after := toGameStateResponse(g); after != before || !g.turnStartedAt.Equal(started) || len(g.historyKeys) != 1 {
		t.Fatal("rejected move changed game state or clock")
	}
}

func TestConcurrentMovesAndReads(t *testing.T) {
	h := NewHTTPHandler()
	state := createTestGame(t, h)
	base := "/api/v1/games/" + state.ID
	const count = 12
	start := make(chan struct{})
	codes := make(chan int, count)
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, base+"/move", strings.NewReader(`{"move":"e2e4"}`)))
			codes <- w.Code
		}()
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 5; j++ {
				w := httptest.NewRecorder()
				h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, base, nil))
				var snapshot gameStateResponse
				if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &snapshot) != nil {
					t.Errorf("invalid GET response: %d %s", w.Code, w.Body.String())
					return
				}
				pos, err := chess.ParseFEN(snapshot.FEN)
				if err != nil || pos.SideToMove.String() != snapshot.SideToMove || !snapshot.YourTurn {
					t.Errorf("inconsistent or intermediate snapshot: %+v", snapshot)
				}
				w = httptest.NewRecorder()
				h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, base+"/legal-moves?from=g1", nil))
				var moves legalMovesResponse
				if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &moves) != nil || len(moves.Moves) == 0 {
					t.Errorf("invalid legal moves response: %d %s", w.Code, w.Body.String())
				}
			}
		}()
	}
	close(start)
	wg.Wait()
	close(codes)
	succeeded := 0
	for code := range codes {
		if code == http.StatusOK {
			succeeded++
		} else if code != http.StatusBadRequest {
			t.Errorf("unexpected move status: %d", code)
		}
	}
	if succeeded != 1 {
		t.Fatalf("expected exactly one accepted move, got %d", succeeded)
	}
	g := storeGet(state.ID)
	g.mu.RLock()
	defer g.mu.RUnlock()
	if len(g.historyKeys) != 3 || g.sideToMove != g.humanSide {
		t.Fatalf("expected one human move and one bot reply, history=%d side=%s", len(g.historyKeys), g.sideToMove)
	}
}

func TestFinishedGameDisallowsMoves(t *testing.T) {
	for _, status := range []string{"checkmate", "stalemate", "draw"} {
		t.Run(status, func(t *testing.T) {
			h := NewHTTPHandler()
			state := createTestGame(t, h)
			g := storeGet(state.ID)
			g.mu.Lock()
			g.status = status
			g.mu.Unlock()
			base := "/api/v1/games/" + state.ID
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, base, nil))
			if err := json.Unmarshal(w.Body.Bytes(), &state); err != nil {
				t.Fatal(err)
			}
			if state.YourTurn {
				t.Fatal("finished game reports yourTurn=true")
			}
			w = httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, base+"/move", strings.NewReader(`{"move":"e2e4"}`)))
			if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "game is finished") {
				t.Fatalf("expected finished game error, got %d %s", w.Code, w.Body.String())
			}
		})
	}
}
