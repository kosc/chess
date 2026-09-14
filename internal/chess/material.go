package chess

// HasInsufficientMatingMaterial identifies material that cannot deliver mate
// for one side, including help from the opponent's pieces. It does not solve
// position-dependent dead positions such as blocked pawn fortresses.
func HasInsufficientMatingMaterial(pos Position, side Color) bool {
	knights, bishops := 0, 0
	var bishopColors [2]bool
	anyPawnOrKnight := false
	opponentCanBlockKnightEscape := false
	for sq, pc := range pos.Board {
		if pc.IsZero() || pc.Type == King {
			continue
		}
		if pc.Type == Bishop {
			bishopColors[squareColorIndex(Square(sq))] = true
		}
		if pc.Type == Pawn || pc.Type == Knight {
			anyPawnOrKnight = true
		}
		if pc.Color != side {
			if pc.Type != Queen {
				opponentCanBlockKnightEscape = true
			}
			continue
		}
		switch pc.Type {
		case Pawn, Rook, Queen:
			return false
		case Knight:
			knights++
		case Bishop:
			bishops++
		}
	}
	if knights > 0 {
		return knights == 1 && bishops == 0 && !opponentCanBlockKnightEscape
	}
	if bishops > 0 {
		return !anyPawnOrKnight && !(bishopColors[0] && bishopColors[1])
	}
	return true
}
