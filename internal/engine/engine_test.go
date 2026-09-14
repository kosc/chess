package engine

import (
	"context"
	"testing"
	"time"

	"github.com/kosc/chessweb/internal/chess"
)

func TestNegamaxAtSearchBoundary(t *testing.T) {
	for _, tc := range []struct {
		name, fen string
		want      int
	}{
		{"black_mated", "7k/6Q1/5K2/8/8/8/8/8 b - - 0 1", -100000},
		{"white_mated", "8/8/8/8/8/5k2/6q1/7K w - - 0 1", -100000},
		{"stalemate", "7k/5Q2/5K2/8/8/8/8/8 b - - 0 1", 0},
		{"ongoing", "7k/8/5KQ1/8/8/8/8/8 w - - 0 1", 900},
		{"check_with_escape", "R6k/8/8/8/8/8/8/K7 b - - 0 1", -500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pos, err := chess.ParseFEN(tc.fen)
			if err != nil {
				t.Fatal(err)
			}
			score, completed := negamax(context.Background(), pos, 0, -9999999, 9999999)
			if !completed || score != tc.want {
				t.Fatalf("score=%d completed=%v, want %d", score, completed, tc.want)
			}
		})
	}
}

func TestSearchFindsMateOnLastPly(t *testing.T) {
	for _, tc := range []struct{ name, fen string }{
		{"white", "7k/8/5KQ1/8/8/8/8/8 w - - 0 1"},
		{"black", "8/8/8/8/8/5kq1/8/7K b - - 0 1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pos, err := chess.ParseFEN(tc.fen)
			if err != nil {
				t.Fatal(err)
			}
			move, score, ok := searchRoot(context.Background(), pos, 1)
			if !ok {
				t.Fatal("search did not return a move")
			}
			next, err := chess.ApplyMove(pos, move)
			if err != nil {
				t.Fatal(err)
			}
			if status := chess.EvaluateStatus(next, nil); status.Status != "checkmate" || score != 100000 {
				t.Fatalf("move %s: status=%s score=%d, want mate", move.UCI(), status.Status, score)
			}
		})
	}
}

func TestBestMoveWithExpiredBudget(t *testing.T) {
	pos, err := chess.ParseFEN(chess.StartFEN())
	if err != nil {
		t.Fatal(err)
	}
	m, ok := (Engine{MaxPly: 4, Think: time.Nanosecond}).BestMove(pos)
	if !ok {
		t.Fatal("time limit must not prevent a legal reply")
	}
	for _, legal := range chess.LegalMovesFrom(pos, m.From) {
		if m == legal {
			return
		}
	}
	t.Fatalf("illegal fallback move: %s", m.UCI())
}

func TestBestMoveWithoutLegalMoves(t *testing.T) {
	for _, fen := range []string{
		"7k/6Q1/5K2/8/8/8/8/8 b - - 0 1",
		"7k/5Q2/5K2/8/8/8/8/8 b - - 0 1",
	} {
		pos, err := chess.ParseFEN(fen)
		if err != nil {
			t.Fatal(err)
		}
		if m, ok := (Engine{Think: time.Nanosecond}).BestMove(pos); ok {
			t.Fatalf("unexpected move %s in terminal position %s", m.UCI(), fen)
		}
	}
}
