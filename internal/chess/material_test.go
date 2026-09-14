package chess

import "testing"

func TestInsufficientMatingMaterial(t *testing.T) {
	for _, tc := range []struct {
		name, fen string
		side      Color
		want      bool
	}{
		{"bare_black_king", "7k/8/8/8/8/8/8/KR6 w - - 0 1", Black, true},
		{"white_rook", "7k/8/8/8/8/8/8/KR6 w - - 0 1", White, false},
		{"knight_vs_king", "7k/8/8/8/8/8/8/KN6 w - - 0 1", White, true},
		{"knight_vs_queen", "6qk/8/8/8/8/8/8/KN6 w - - 0 1", White, true},
		{"knight_vs_pawn", "7k/7p/8/8/8/8/8/KN6 w - - 0 1", White, false},
		{"knight_vs_rook", "6rk/8/8/8/8/8/8/KN6 w - - 0 1", White, false},
		{"two_knights", "7k/8/8/8/8/8/8/KNN5 w - - 0 1", White, false},
		{"bishop_vs_king", "7k/8/8/8/8/8/8/KB6 w - - 0 1", White, true},
		{"same_color_bishops", "7k/8/8/8/8/8/8/KB1B4 w - - 0 1", White, true},
		{"opposite_color_bishops", "7k/8/8/8/8/8/8/KBB5 w - - 0 1", White, false},
		{"bishop_vs_knight", "6nk/8/8/8/8/8/8/KB6 w - - 0 1", White, false},
		{"bishop_and_knight", "7k/8/8/8/8/8/8/KBN5 w - - 0 1", White, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pos, err := ParseFEN(tc.fen)
			if err != nil {
				t.Fatal(err)
			}
			if got := HasInsufficientMatingMaterial(pos, tc.side); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
