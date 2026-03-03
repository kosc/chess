package chess

// isSquareAttacked checks if `sq` is attacked by color `by`.
func isSquareAttacked(pos Position, sq Square, by Color) bool {
	if !sq.IsValid() {
		return false
	}

	// Pawns
	if pawnAttacksSquare(pos, sq, by) {
		return true
	}

	// Knights
	if knightAttacksSquare(pos, sq, by) {
		return true
	}

	// King (adjacent squares)
	if kingAttacksSquare(pos, sq, by) {
		return true
	}

	// Sliders: bishops/queens diagonals
	if sliderAttacksSquare(pos, sq, by, [][2]int{{1, 1}, {-1, 1}, {1, -1}, {-1, -1}}, Bishop, Queen) {
		return true
	}

	// Sliders: rooks/queens orthogonals
	if sliderAttacksSquare(pos, sq, by, [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}, Rook, Queen) {
		return true
	}

	return false
}

func pawnAttacksSquare(pos Position, sq Square, by Color) bool {
	f, r := sq.File(), sq.Rank()
	if by == White {
		// white pawn attacks from (f-1,r-1) and (f+1,r-1)
		for _, df := range []int{-1, 1} {
			from := SquareFromFR(f+df, r-1)
			if from.IsValid() {
				pc := pos.Board[from]
				if pc.Type == Pawn && pc.Color == White {
					return true
				}
			}
		}
	} else {
		// black pawn attacks from (f-1,r+1) and (f+1,r+1)
		for _, df := range []int{-1, 1} {
			from := SquareFromFR(f+df, r+1)
			if from.IsValid() {
				pc := pos.Board[from]
				if pc.Type == Pawn && pc.Color == Black {
					return true
				}
			}
		}
	}
	return false
}

func knightAttacksSquare(pos Position, sq Square, by Color) bool {
	deltas := [][2]int{
		{1, 2}, {2, 1}, {2, -1}, {1, -2},
		{-1, -2}, {-2, -1}, {-2, 1}, {-1, 2},
	}
	f, r := sq.File(), sq.Rank()
	for _, d := range deltas {
		from := SquareFromFR(f+d[0], r+d[1])
		if !from.IsValid() {
			continue
		}
		pc := pos.Board[from]
		if pc.Type == Knight && pc.Color == by {
			return true
		}
	}
	return false
}

func kingAttacksSquare(pos Position, sq Square, by Color) bool {
	deltas := [][2]int{
		{-1, -1}, {0, -1}, {1, -1},
		{-1, 0}, {1, 0},
		{-1, 1}, {0, 1}, {1, 1},
	}
	f, r := sq.File(), sq.Rank()
	for _, d := range deltas {
		from := SquareFromFR(f+d[0], r+d[1])
		if !from.IsValid() {
			continue
		}
		pc := pos.Board[from]
		if pc.Type == King && pc.Color == by {
			return true
		}
	}
	return false
}

func sliderAttacksSquare(pos Position, sq Square, by Color, dirs [][2]int, t1, t2 PieceType) bool {
	f0, r0 := sq.File(), sq.Rank()
	for _, d := range dirs {
		f, r := f0+d[0], r0+d[1]
		for {
			cur := SquareFromFR(f, r)
			if !cur.IsValid() {
				break
			}
			pc := pos.Board[cur]
			if pc.IsZero() {
				f += d[0]
				r += d[1]
				continue
			}
			if pc.Color == by && (pc.Type == t1 || pc.Type == t2) {
				return true
			}
			break
		}
	}
	return false
}
