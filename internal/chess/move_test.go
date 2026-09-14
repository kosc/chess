package chess

import (
	"fmt"
	"testing"
)

func TestParseUCI(t *testing.T) {
	m, err := ParseUCI("e2e4")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if m.From.String() != "e2" || m.To.String() != "e4" {
		t.Fatalf("got %v", m)
	}

	mp, err := ParseUCI("e7e8q")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if mp.Flags&FlagPromotion == 0 || mp.Promo != Queen {
		t.Fatalf("expected promotion to queen: %+v", mp)
	}
}

func TestCastlingMoveCounters(t *testing.T) {
	for _, tc := range []struct {
		uci, side string
		fullmove  int
	}{
		{"e1g1", "w", 12},
		{"e1c1", "w", 12},
		{"e8g8", "b", 13},
		{"e8c8", "b", 13},
	} {
		for _, halfmove := range []int{98, 99} {
			t.Run(fmt.Sprintf("%s/halfmove_%d", tc.uci, halfmove), func(t *testing.T) {
				pos, err := ParseFEN(fmt.Sprintf("r3k2r/8/8/8/8/8/8/R3K2R %s KQkq - %d 12", tc.side, halfmove))
				if err != nil {
					t.Fatal(err)
				}
				from, err := ParseSquare(tc.uci[:2])
				if err != nil {
					t.Fatal(err)
				}
				var castle Move
				found := false
				for _, m := range LegalMovesFrom(pos, from) {
					if m.UCI() == tc.uci {
						castle, found = m, true
						break
					}
				}
				if !found || castle.Flags&FlagCastle == 0 {
					t.Fatal("expected a legal castling move")
				}
				next, err := ApplyMove(pos, castle)
				if err != nil {
					t.Fatal(err)
				}
				if next.HalfmoveClock != halfmove+1 {
					t.Errorf("halfmove clock: got %d, want %d", next.HalfmoveClock, halfmove+1)
				}
				if next.FullmoveNumber != tc.fullmove || next.SideToMove != pos.SideToMove.Opp() {
					t.Errorf("incorrect move number or side: %s", next.FEN())
				}
				status := EvaluateStatus(next, nil)
				if halfmove == 98 && status.Status != "in_progress" {
					t.Errorf("premature game end after castling: %+v", status)
				}
				if halfmove == 99 && (status.Status != "draw" || status.DrawReason != "fifty_move") {
					t.Errorf("expected fifty-move draw: %+v", status)
				}
			})
		}
	}
}
