package chess

type GameStatus struct {
	Status     string // in_progress | check | checkmate | stalemate | draw
	DrawReason string // threefold | fifty_move | insufficient_material
}

// EvaluateStatus computes status for the side to move in `pos`.
func EvaluateStatus(pos Position, history []string) GameStatus {
	// history: list of position keys for repetition (we'll use FEN without move clocks)
	// For v1: pass from server store.

	// 1) Draw by 50-move rule
	if pos.HalfmoveClock >= 100 {
		return GameStatus{Status: "draw", DrawReason: "fifty_move"}
	}

	// 2) Draw by insufficient material
	if isInsufficientMaterial(pos) {
		return GameStatus{Status: "draw", DrawReason: "insufficient_material"}
	}

	// 3) Draw by threefold repetition
	if isThreefoldRepetition(history, repetitionKey(pos)) {
		return GameStatus{Status: "draw", DrawReason: "threefold"}
	}

	// Check / mate / stalemate
	stm := pos.SideToMove
	kingSq := findKing(pos, stm)
	inCheck := kingSq != NoSquare && isSquareAttacked(pos, kingSq, stm.Opp())

	// Any legal move?
	hasMove := false
	for sq := Square(0); sq < 64; sq++ {
		pc := pos.Board[sq]
		if pc.IsZero() || pc.Color != stm {
			continue
		}
		if len(LegalMovesFrom(pos, sq)) > 0 {
			hasMove = true
			break
		}
	}

	if !hasMove {
		if inCheck {
			return GameStatus{Status: "checkmate"}
		}
		return GameStatus{Status: "stalemate"}
	}

	if inCheck {
		return GameStatus{Status: "check"}
	}
	return GameStatus{Status: "in_progress"}
}

// repetitionKey excludes halfmove and fullmove counters.
func repetitionKey(pos Position) string {
	// fields: placement stm castling ep
	// We'll reuse pos.FEN() and cut last two fields.
	f := pos.FEN()
	// FEN has 6 fields, cut after 4th
	// Simple split is fine for v1.
	fields := splitFields6(f)
	return fields[0] + " " + fields[1] + " " + fields[2] + " " + fields[3]
}

func splitFields6(fen string) [6]string {
	var out [6]string
	i := 0
	start := 0
	for j := 0; j <= len(fen) && i < 6; j++ {
		if j == len(fen) || fen[j] == ' ' {
			out[i] = fen[start:j]
			i++
			start = j + 1
		}
	}
	return out
}

func isThreefoldRepetition(history []string, key string) bool {
	n := 0
	for _, k := range history {
		if k == key {
			n++
		}
	}
	return n >= 3
}

func isInsufficientMaterial(pos Position) bool {
	// Very common minimal set:
	// K vs K
	// K vs K+B
	// K vs K+N
	// K+B vs K+B (same color bishops) — optional; we’ll implement basic subset first, then extend

	// Count material excluding kings
	var wMinor, bMinor int
	var wBishOn [2]int // light/dark bishops count (for possible extension)
	var bBishOn [2]int
	var wOther, bOther int

	for sq := Square(0); sq < 64; sq++ {
		pc := pos.Board[sq]
		if pc.IsZero() || pc.Type == King {
			continue
		}
		switch pc.Type {
		case Pawn, Rook, Queen:
			if pc.Color == White {
				wOther++
			} else {
				bOther++
			}
		case Bishop:
			if pc.Color == White {
				wMinor++
				wBishOn[squareColorIndex(sq)]++
			} else {
				bMinor++
				bBishOn[squareColorIndex(sq)]++
			}
		case Knight:
			if pc.Color == White {
				wMinor++
			} else {
				bMinor++
			}
		}
	}

	if wOther > 0 || bOther > 0 {
		return false
	}

	// K vs K
	if wMinor == 0 && bMinor == 0 {
		return true
	}

	// K+minor vs K
	if (wMinor == 1 && bMinor == 0) || (wMinor == 0 && bMinor == 1) {
		return true
	}

	// Optional extension: K+B vs K+B same color bishops
	// We'll include it for correctness.
	if wMinor == 1 && bMinor == 1 {
		// check both are bishops
		wIsB := (wBishOn[0]+wBishOn[1] == 1)
		bIsB := (bBishOn[0]+bBishOn[1] == 1)
		if wIsB && bIsB {
			// same-color bishops -> draw
			if (wBishOn[0] == 1 && bBishOn[0] == 1) || (wBishOn[1] == 1 && bBishOn[1] == 1) {
				return true
			}
		}
	}

	return false
}

// returns 0 for light, 1 for dark
func squareColorIndex(sq Square) int {
	// a1 is dark in standard? Actually a1 is dark (black). We'll just compute parity.
	// light/dark for bishops equality doesn't depend on naming, only parity.
	parity := (sq.File() + sq.Rank()) % 2
	if parity == 0 {
		return 0
	}
	return 1
}
