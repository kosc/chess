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
	return createTestGameWithRequest(t, h, createGameRequest{BotMaxPly: 2, BotMoveTimeMs: 5000})
}

func createTestGameWithRequest(t *testing.T, h http.Handler, req createGameRequest) gameStateResponse {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/games", strings.NewReader(string(body))))
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

func TestCreateGameEvaluatesInitialPosition(t *testing.T) {
	for _, tc := range []struct {
		name, fen, status, reason string
	}{
		{"mate", "7k/6Q1/5K2/8/8/8/8/8 b - - 0 1", "checkmate", ""},
		{"stalemate", "7k/5Q2/5K2/8/8/8/8/8 b - - 0 1", "stalemate", ""},
		{"material", "7k/8/5K2/8/8/8/8/8 b - - 0 1", "draw", "insufficient_material"},
		{"fifty_move", "7k/8/5K2/8/8/8/8/R7 b - - 100 1", "draw", "fifty_move"},
	} {
		for _, side := range []string{"white", "black"} {
			t.Run(tc.name+"/"+side, func(t *testing.T) {
				h := NewHTTPHandler()
				state := createTestGameWithRequest(t, h, createGameRequest{FEN: tc.fen, HumanSide: side})
				if state.Status != tc.status || state.DrawReason != tc.reason || state.YourTurn || state.LastMove != "" || state.FEN != tc.fen {
					t.Fatalf("incorrect initial state: %+v", state)
				}
				w := httptest.NewRecorder()
				h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/games/"+state.ID+"/move", strings.NewReader(`{"move":"h8h7"}`)))
				if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "game is finished") {
					t.Fatalf("terminal position accepted move: %d %s", w.Code, w.Body.String())
				}
			})
		}
	}
	t.Run("check", func(t *testing.T) {
		state := createTestGameWithRequest(t, NewHTTPHandler(), createGameRequest{
			FEN: "R6k/8/8/8/8/8/8/K7 b - - 0 1", HumanSide: "black",
		})
		if state.Status != "check" || !state.YourTurn || state.LastMove != "" {
			t.Fatalf("incorrect check state: %+v", state)
		}
	})
}

func TestHumanMoveAndBotReply(t *testing.T) {
	h := NewHTTPHandler()
	state := createTestGameWithRequest(t, h, createGameRequest{BotMaxPly: 4, BotMoveTimeMs: 1})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/games/"+state.ID+"/move", strings.NewReader(`{"move":"e2e4"}`)))
	if w.Code != http.StatusOK {
		t.Fatalf("move: %d %s", w.Code, w.Body.String())
	}
	var next gameStateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &next); err != nil {
		t.Fatal(err)
	}
	pos, _ := chess.ParseFEN(state.FEN)
	from, _ := chess.ParseSquare("e2")
	human := matchLegalMove(chess.LegalMovesFrom(pos, from), "e2e4")
	if human == nil {
		t.Fatal("missing human move")
	}
	afterHuman, err := chess.ApplyMove(pos, *human)
	if err != nil {
		t.Fatal(err)
	}
	bot, err := chess.ParseUCI(next.LastMove)
	if err != nil {
		t.Fatalf("missing bot reply: %+v", next)
	}
	legalBot := matchLegalMove(chess.LegalMovesFrom(afterHuman, bot.From), next.LastMove)
	if legalBot == nil {
		t.Fatalf("illegal bot reply: %s", next.LastMove)
	}
	expected, err := chess.ApplyMove(afterHuman, *legalBot)
	if err != nil || next.FEN != expected.FEN() || !next.YourTurn || next.SideToMove != "white" {
		t.Fatalf("incorrect state after bot reply: %+v", next)
	}
}

func TestIllegalMoveLeavesPositionUnchanged(t *testing.T) {
	h := NewHTTPHandler()
	state := createTestGame(t, h)
	base := "/api/v1/games/" + state.ID
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, base+"/move", strings.NewReader(`{"move":"e2e5"}`)))
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "illegal move") {
		t.Fatalf("expected illegal move error: %d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, base, nil))
	var after gameStateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &after); err != nil {
		t.Fatal(err)
	}
	if after != state {
		t.Fatalf("illegal move changed state: %+v", after)
	}
}

func TestHumanMoveEndsGameWithoutBotReply(t *testing.T) {
	for _, tc := range []struct{ name, fen, move, status string }{
		{"mate", "7k/8/5KQ1/8/8/8/8/8 w - - 0 1", "g6g7", "checkmate"},
		{"stalemate", "7k/8/5K2/6Q1/8/8/8/8 w - - 0 1", "g5g6", "stalemate"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHTTPHandler()
			state := createTestGameWithRequest(t, h, createGameRequest{FEN: tc.fen})
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/games/"+state.ID+"/move", strings.NewReader(`{"move":"`+tc.move+`"}`)))
			if w.Code != http.StatusOK {
				t.Fatalf("move: %d %s", w.Code, w.Body.String())
			}
			if err := json.Unmarshal(w.Body.Bytes(), &state); err != nil {
				t.Fatal(err)
			}
			if state.Status != tc.status || state.LastMove != tc.move || state.YourTurn || state.SideToMove != "black" {
				t.Fatalf("incorrect terminal state: %+v", state)
			}
			g := storeGet(state.ID)
			g.mu.RLock()
			defer g.mu.RUnlock()
			if len(g.historyKeys) != 2 {
				t.Fatalf("expected only human move, history length=%d", len(g.historyKeys))
			}
		})
	}
}
