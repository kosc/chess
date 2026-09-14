package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kosc/chessweb/internal/chess"
)

func TestRemainingSeconds(t *testing.T) {
	for _, tc := range []struct {
		remaining time.Duration
		want      int
	}{
		{-time.Second, 0}, {0, 0}, {time.Nanosecond, 1},
		{time.Second, 1}, {1500 * time.Millisecond, 2},
	} {
		if got := remainingSeconds(tc.remaining); got != tc.want {
			t.Errorf("remaining=%s: got %d, want %d", tc.remaining, got, tc.want)
		}
	}
}

func TestClockRetainsFractionsAcrossRequestsAndTurns(t *testing.T) {
	start := time.Unix(1000, 0)
	g := &game{fen: chess.StartFEN(), status: "in_progress", sideToMove: "white",
		clockEnabled: true, whiteRemaining: time.Second, blackRemaining: time.Second, turnStartedAt: start}
	for i := 1; i <= 3; i++ {
		if g.updateClock(start.Add(time.Duration(i) * 100 * time.Millisecond)) {
			t.Fatal("premature timeout")
		}
	}
	g.sideToMove = "black"
	g.updateClock(start.Add(550 * time.Millisecond))
	g.sideToMove = "white"
	g.updateClock(start.Add(850 * time.Millisecond))
	if g.whiteRemaining != 400*time.Millisecond || g.blackRemaining != 750*time.Millisecond {
		t.Fatalf("lost fractions: white=%s black=%s", g.whiteRemaining, g.blackRemaining)
	}
	if !g.updateClock(start.Add(1250*time.Millisecond)) || g.status != "timeout" || g.winner != "black" {
		t.Fatal("expected white timeout exactly at zero")
	}
	if g.whiteRemaining != 0 || g.blackRemaining != 750*time.Millisecond {
		t.Fatal("incorrect remaining time after timeout")
	}
	g.updateClock(start.Add(time.Hour))
	if g.blackRemaining != 750*time.Millisecond || g.winner != "black" {
		t.Fatal("finished clocks continued running")
	}
}

func TestClockDisabledOrFinished(t *testing.T) {
	for _, status := range []string{"disabled", "checkmate", "stalemate", "draw", "timeout"} {
		t.Run(status, func(t *testing.T) {
			start := time.Unix(1000, 0)
			g := &game{fen: chess.StartFEN(), status: status, sideToMove: "white", clockEnabled: status != "disabled",
				whiteRemaining: time.Second, blackRemaining: time.Second, turnStartedAt: start}
			if status == "disabled" {
				g.status = "in_progress"
			}
			g.updateClock(start.Add(time.Hour))
			if g.whiteRemaining != time.Second || g.blackRemaining != time.Second || g.turnStartedAt != start {
				t.Fatal("inactive clock changed")
			}
		})
	}
}

func TestClockExpiryThroughAPI(t *testing.T) {
	for _, side := range []string{"white", "black"} {
		for _, route := range []string{"get", "move", "legal-moves"} {
			t.Run(side+"/"+route, func(t *testing.T) {
				h := NewHTTPHandler()
				fen, move, winner := chess.StartFEN(), "e2e4", "black"
				if side == "black" {
					fen, move, winner = strings.Replace(fen, " w ", " b ", 1), "e7e5", "white"
				}
				state := createTestGameWithRequest(t, h, createGameRequest{FEN: fen, HumanSide: side, ClockEnabled: true, InitialSeconds: 1})
				g := storeGet(state.ID)
				g.mu.Lock()
				g.turnStartedAt = time.Now().Add(-2 * time.Second)
				g.mu.Unlock()
				base := "/api/v1/games/" + state.ID
				method, path, body := http.MethodGet, base, ""
				if route == "move" {
					method, path, body = http.MethodPost, base+"/move", `{"move":"`+move+`"}`
				} else if route == "legal-moves" {
					method, path = http.MethodPost, base+"/legal-moves?from="+move[:2]
				}
				w := httptest.NewRecorder()
				h.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
				if w.Code != http.StatusOK {
					t.Fatalf("unexpected status: %d %s", w.Code, w.Body.String())
				}
				if route == "legal-moves" {
					var moves legalMovesResponse
					if err := json.Unmarshal(w.Body.Bytes(), &moves); err != nil || len(moves.Moves) != 0 || len(moves.ToSquares) != 0 {
						t.Fatal("expired game offers moves")
					}
				}
				w = httptest.NewRecorder()
				h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, base, nil))
				var after gameStateResponse
				if err := json.Unmarshal(w.Body.Bytes(), &after); err != nil {
					t.Fatal(err)
				}
				if after.Status != "timeout" || after.Winner != winner || after.DrawReason != "" || after.YourTurn || after.FEN != state.FEN || after.LastMove != "" {
					t.Fatalf("wrong timeout result: %+v", after)
				}
				var raw map[string]any
				if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil || raw[side+"Sec"] != float64(0) {
					t.Fatalf("zero clock omitted: %s", w.Body.String())
				}
			})
		}
	}
}

func TestBotTimeoutDoesNotApplyBotMove(t *testing.T) {
	h := NewHTTPHandler()
	state := createTestGameWithRequest(t, h, createGameRequest{ClockEnabled: true, BotMaxPly: 2, BotMoveTimeMs: 1})
	g := storeGet(state.ID)
	g.mu.Lock()
	g.blackRemaining = 0
	g.mu.Unlock()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/games/"+state.ID+"/move", strings.NewReader(`{"move":"e2e4"}`)))
	var after gameStateResponse
	if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &after) != nil {
		t.Fatalf("invalid move response: %s", w.Body.String())
	}
	if after.Status != "timeout" || after.Winner != "white" || after.LastMove != "e2e4" || after.SideToMove != "black" || after.BlackSec != 0 {
		t.Fatalf("wrong bot timeout: %+v", after)
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	if len(g.historyKeys) != 2 {
		t.Fatal("bot moved after timeout")
	}
}

func TestTimeoutWithInsufficientOpponentMaterial(t *testing.T) {
	h := NewHTTPHandler()
	state := createTestGameWithRequest(t, h, createGameRequest{
		FEN: "7k/8/8/8/8/8/8/KR6 w - - 0 1", ClockEnabled: true, InitialSeconds: 1,
	})
	g := storeGet(state.ID)
	g.mu.Lock()
	g.turnStartedAt = time.Now().Add(-2 * time.Second)
	g.mu.Unlock()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/games/"+state.ID, nil))
	if err := json.Unmarshal(w.Body.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	if state.Status != "draw" || state.DrawReason != "timeout_insufficient_material" || state.Winner != "" || state.YourTurn {
		t.Fatalf("bare king cannot win on time: %+v", state)
	}
}

func TestIllegalRequestsStillConsumeFractions(t *testing.T) {
	h := NewHTTPHandler()
	state := createTestGameWithRequest(t, h, createGameRequest{ClockEnabled: true, InitialSeconds: 10})
	g := storeGet(state.ID)
	for i := 0; i < 5; i++ {
		g.mu.Lock()
		before := g.whiteRemaining
		g.turnStartedAt = time.Now().Add(-200 * time.Millisecond)
		g.mu.Unlock()
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/games/"+state.ID+"/move", strings.NewReader(`{"move":"e2e5"}`)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("illegal move status: %d", w.Code)
		}
		g.mu.RLock()
		after := g.whiteRemaining
		g.mu.RUnlock()
		if before-after < 200*time.Millisecond {
			t.Fatal("request discarded elapsed fraction")
		}
	}
}
