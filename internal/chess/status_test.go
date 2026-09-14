package chess

import (
	"fmt"
	"testing"
)

func TestTerminalStatusAtFiftyMoveBoundary(t *testing.T) {
	for _, tc := range []struct {
		name, placement, side, status string
	}{
		{"black_mated", "7k/6Q1/5K2/8/8/8/8/8", "b", "checkmate"},
		{"white_mated", "8/8/8/8/8/5k2/6q1/7K", "w", "checkmate"},
		{"stalemate", "7k/5Q2/5K2/8/8/8/8/8", "b", "stalemate"},
	} {
		for _, halfmove := range []int{99, 100, 101} {
			t.Run(fmt.Sprintf("%s/%d", tc.name, halfmove), func(t *testing.T) {
				pos, err := ParseFEN(fmt.Sprintf("%s %s - - %d 60", tc.placement, tc.side, halfmove))
				if err != nil {
					t.Fatal(err)
				}
				if got := EvaluateStatus(pos, nil); got.Status != tc.status || got.DrawReason != "" {
					t.Fatalf("got %+v, want %s without draw reason", got, tc.status)
				}
			})
		}
	}
}

func TestFiftyMoveDrawWithLegalMoves(t *testing.T) {
	for _, tc := range []struct{ name, placement, status string }{
		{"not_in_check", "7k/8/8/8/8/8/8/KR6", "in_progress"},
		{"in_check", "R6k/8/8/8/8/8/8/K7", "check"},
	} {
		for _, halfmove := range []int{99, 100} {
			t.Run(fmt.Sprintf("%s/%d", tc.name, halfmove), func(t *testing.T) {
				pos, err := ParseFEN(fmt.Sprintf("%s b - - %d 60", tc.placement, halfmove))
				if err != nil {
					t.Fatal(err)
				}
				want := GameStatus{Status: tc.status}
				if halfmove == 100 {
					want = GameStatus{Status: "draw", DrawReason: "fifty_move"}
				}
				if got := EvaluateStatus(pos, nil); got != want {
					t.Fatalf("got %+v, want %+v", got, want)
				}
			})
		}
	}
}
