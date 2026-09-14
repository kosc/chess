package engine

import (
	"testing"
	"time"

	"github.com/kosc/chessweb/internal/chess"
)

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
