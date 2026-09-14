package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/kosc/chessweb/internal/chess"
)

func TestMoveHistory(t *testing.T) {
	for _, tc := range []struct {
		name, fen, side, move, recorded string
		number                          int
	}{
		{"normal", chess.StartFEN(), "white", "e2e4", "e2e4", 1},
		{"black_from_fen", "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR b KQkq - 0 17", "black", "e7e5", "e7e5", 17},
		{"castle", "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 12", "white", "e1g1", "e1g1", 12},
		{"promotion", "7k/P7/8/8/8/8/8/K7 w - - 0 9", "white", "a7a8", "a7a8q", 9},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHTTPHandler()
			state := createTestGameWithRequest(t, h, createGameRequest{FEN: tc.fen, HumanSide: tc.side, BotMaxPly: 2, BotMoveTimeMs: 100})
			if state.MoveHistory == nil || len(state.MoveHistory) != 0 {
				t.Fatalf("new game must return an empty array: %+v", state.MoveHistory)
			}
			base := "/api/v1/games/" + state.ID
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, base+"/move", strings.NewReader(`{"move":"`+tc.move+`"}`)))
			if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &state) != nil {
				t.Fatalf("move failed: %d %s", w.Code, w.Body.String())
			}
			if len(state.MoveHistory) != 2 || state.MoveHistory[0] != (playedMove{Number: tc.number, Side: tc.side, UCI: tc.recorded}) {
				t.Fatalf("incorrect history: %+v", state.MoveHistory)
			}
			if tc.name == "normal" {
				previous := append([]playedMove{}, state.MoveHistory...)
				w = httptest.NewRecorder()
				h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, base+"/move", strings.NewReader(`{"move":"g1f3"}`)))
				if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &state) != nil {
					t.Fatalf("second move failed: %s", w.Body.String())
				}
				if len(state.MoveHistory) != 4 || !reflect.DeepEqual(state.MoveHistory[:2], previous) {
					t.Fatalf("history was not appended: %+v", state.MoveHistory)
				}
			}
			pos, err := chess.ParseFEN(tc.fen)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range state.MoveHistory {
				if entry.Number != pos.FullmoveNumber || entry.Side != pos.SideToMove.String() {
					t.Fatalf("incorrect move number or side: %+v", entry)
				}
				move, err := chess.ParseUCI(entry.UCI)
				if err != nil {
					t.Fatal(err)
				}
				legal := matchLegalMove(chess.LegalMovesFrom(pos, move.From), entry.UCI)
				if legal == nil {
					t.Fatalf("illegal history move: %+v", entry)
				}
				pos, err = chess.ApplyMove(pos, *legal)
				if err != nil {
					t.Fatal(err)
				}
			}
			if pos.FEN() != state.FEN || state.LastMove != state.MoveHistory[len(state.MoveHistory)-1].UCI {
				t.Fatal("history does not reproduce the current position")
			}
			w = httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, base, nil))
			var restored gameStateResponse
			if err := json.Unmarshal(w.Body.Bytes(), &restored); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(restored.MoveHistory, state.MoveHistory) {
				t.Fatal("GET did not restore the full history")
			}
		})
	}
}

func TestHistoryIncludesBotOpening(t *testing.T) {
	state := createTestGameWithRequest(t, NewHTTPHandler(), createGameRequest{HumanSide: "black"})
	if len(state.MoveHistory) != 1 || state.MoveHistory[0] != (playedMove{Number: 1, Side: "white", UCI: state.LastMove}) {
		t.Fatalf("missing bot opening: %+v", state.MoveHistory)
	}
}

func TestHistorySnapshotIsIndependent(t *testing.T) {
	g := &game{moves: []playedMove{{Number: 1, Side: "white", UCI: "e2e4"}}}
	response := toGameStateResponse(g)
	response.MoveHistory[0].UCI = "d2d4"
	if g.moves[0].UCI != "e2e4" {
		t.Fatal("response aliases game history")
	}
}
