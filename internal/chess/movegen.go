package chess

import "errors"

// LegalMovesFrom returns legal moves for the side to move that originate from `from`.
// If `from` is empty or not occupied by side-to-move piece, returns empty.
func LegalMovesFrom(pos Position, from Square) []Move {
	if !from.IsValid() {
		return nil
	}
	pc := pos.Board[from]
	if pc.IsZero() || pc.Color != pos.SideToMove {
		return nil
	}

	pseudo := pseudoMovesFrom(pos, from)
	out := make([]Move, 0, len(pseudo))
	for _, m := range pseudo {
		if isLegalMove(pos, m) {
			out = append(out, m)
		}
	}
	return out
}

func isLegalMove(pos Position, m Move) bool {
	if !m.IsValid() {
		return false
	}
	pc := pos.Board[m.From]
	if pc.IsZero() || pc.Color != pos.SideToMove {
		return false
	}

	// For castling: ensure squares are not attacked (king passes through) and rook exists etc.
	// We'll validate details inside ApplyMove by simulating; but we must check "through check" here.
	if m.Flags&FlagCastle != 0 {
		return isLegalCastle(pos, m)
	}

	// Simulate move
	next, err := ApplyMove(pos, m)
	if err != nil {
		return false
	}

	// King cannot be in check after move
	kingSq := findKing(next, pc.Color)
	if kingSq == NoSquare {
		return false
	}
	if isSquareAttacked(next, kingSq, pc.Color.Opp()) {
		return false
	}
	return true
}

func isLegalCastle(pos Position, m Move) bool {
	pc := pos.Board[m.From]
	if pc.Type != King {
		return false
	}

	// Must not be in check now
	kingSq := m.From
	if isSquareAttacked(pos, kingSq, pc.Color.Opp()) {
		return false
	}

	// Determine pass-through squares
	// White: e1->g1 (passes f1), e1->c1 (passes d1)
	// Black: e8->g8 (passes f8), e8->c8 (passes d8)
	var through []Square
	if pc.Color == White && m.From == mustSq("e1") && m.To == mustSq("g1") {
		through = []Square{mustSq("f1"), mustSq("g1")}
	} else if pc.Color == White && m.From == mustSq("e1") && m.To == mustSq("c1") {
		through = []Square{mustSq("d1"), mustSq("c1")}
	} else if pc.Color == Black && m.From == mustSq("e8") && m.To == mustSq("g8") {
		through = []Square{mustSq("f8"), mustSq("g8")}
	} else if pc.Color == Black && m.From == mustSq("e8") && m.To == mustSq("c8") {
		through = []Square{mustSq("d8"), mustSq("c8")}
	} else {
		return false
	}

	for _, sq := range through {
		if isSquareAttacked(pos, sq, pc.Color.Opp()) {
			return false
		}
	}

	// Also rely on ApplyMove to validate emptiness, rook, rights.
	_, err := ApplyMove(pos, m)
	return err == nil
}

func pseudoMovesFrom(pos Position, from Square) []Move {
	pc := pos.Board[from]
	switch pc.Type {
	case Pawn:
		return pseudoPawnMoves(pos, from, pc.Color)
	case Knight:
		return pseudoKnightMoves(pos, from, pc.Color)
	case Bishop:
		return pseudoSliderMoves(pos, from, pc.Color, [][2]int{{1, 1}, {-1, 1}, {1, -1}, {-1, -1}})
	case Rook:
		return pseudoSliderMoves(pos, from, pc.Color, [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}})
	case Queen:
		return pseudoSliderMoves(pos, from, pc.Color, [][2]int{{1, 1}, {-1, 1}, {1, -1}, {-1, -1}, {1, 0}, {-1, 0}, {0, 1}, {0, -1}})
	case King:
		return pseudoKingMoves(pos, from, pc.Color)
	default:
		return nil
	}
}

func pseudoKnightMoves(pos Position, from Square, c Color) []Move {
	deltas := [][2]int{
		{1, 2}, {2, 1}, {2, -1}, {1, -2},
		{-1, -2}, {-2, -1}, {-2, 1}, {-1, 2},
	}
	ff, rr := from.File(), from.Rank()
	out := make([]Move, 0, 8)
	for _, d := range deltas {
		to := SquareFromFR(ff+d[0], rr+d[1])
		if !to.IsValid() {
			continue
		}
		dst := pos.Board[to]
		if !dst.IsZero() && dst.Color == c {
			continue
		}
		m := Move{From: from, To: to}
		if !dst.IsZero() {
			m.Flags |= FlagCapture
		}
		out = append(out, m)
	}
	return out
}

func pseudoSliderMoves(pos Position, from Square, c Color, dirs [][2]int) []Move {
	ff, rr := from.File(), from.Rank()
	out := make([]Move, 0, 16)
	for _, d := range dirs {
		f, r := ff+d[0], rr+d[1]
		for {
			to := SquareFromFR(f, r)
			if !to.IsValid() {
				break
			}
			dst := pos.Board[to]
			if dst.IsZero() {
				out = append(out, Move{From: from, To: to})
			} else {
				if dst.Color != c {
					out = append(out, Move{From: from, To: to, Flags: FlagCapture})
				}
				break
			}
			f += d[0]
			r += d[1]
		}
	}
	return out
}

func pseudoKingMoves(pos Position, from Square, c Color) []Move {
	deltas := [][2]int{
		{-1, -1}, {0, -1}, {1, -1},
		{-1, 0}, {1, 0},
		{-1, 1}, {0, 1}, {1, 1},
	}
	ff, rr := from.File(), from.Rank()
	out := make([]Move, 0, 10)
	for _, d := range deltas {
		to := SquareFromFR(ff+d[0], rr+d[1])
		if !to.IsValid() {
			continue
		}
		dst := pos.Board[to]
		if !dst.IsZero() && dst.Color == c {
			continue
		}
		m := Move{From: from, To: to}
		if !dst.IsZero() {
			m.Flags |= FlagCapture
		}
		out = append(out, m)
	}

	// Castling pseudo (legal checked later)
	if c == White && from == mustSq("e1") {
		if pos.Castling&CastleWK != 0 {
			out = append(out, Move{From: from, To: mustSq("g1"), Flags: FlagCastle})
		}
		if pos.Castling&CastleWQ != 0 {
			out = append(out, Move{From: from, To: mustSq("c1"), Flags: FlagCastle})
		}
	}
	if c == Black && from == mustSq("e8") {
		if pos.Castling&CastleBK != 0 {
			out = append(out, Move{From: from, To: mustSq("g8"), Flags: FlagCastle})
		}
		if pos.Castling&CastleBQ != 0 {
			out = append(out, Move{From: from, To: mustSq("c8"), Flags: FlagCastle})
		}
	}
	return out
}

func pseudoPawnMoves(pos Position, from Square, c Color) []Move {
	ff, rr := from.File(), from.Rank()
	out := make([]Move, 0, 8)

	dir := 1
	startRank := 1
	promoRank := 7
	epCaptureRank := 4
	if c == Black {
		dir = -1
		startRank = 6
		promoRank = 0
		epCaptureRank = 3
	}

	// one step
	fwd := SquareFromFR(ff, rr+dir)
	if fwd.IsValid() && pos.Board[fwd].IsZero() {
		addPawnAdvance(&out, from, fwd, c, rr+dir == promoRank, false)

		// two step
		if rr == startRank {
			fwd2 := SquareFromFR(ff, rr+2*dir)
			if fwd2.IsValid() && pos.Board[fwd2].IsZero() {
				out = append(out, Move{From: from, To: fwd2, Flags: FlagDoublePawnPush})
			}
		}
	}

	// captures
	for _, df := range []int{-1, 1} {
		to := SquareFromFR(ff+df, rr+dir)
		if !to.IsValid() {
			continue
		}
		dst := pos.Board[to]
		if !dst.IsZero() && dst.Color != c {
			addPawnAdvance(&out, from, to, c, rr+dir == promoRank, true)
		}

		// en passant capture: target square is pos.EnPassant
		if rr == epCaptureRank && pos.EnPassant == to {
			out = append(out, Move{From: from, To: to, Flags: FlagEnPassant | FlagCapture})
		}
	}

	return out
}

func addPawnAdvance(out *[]Move, from, to Square, c Color, isPromo bool, isCapture bool) {
	if !isPromo {
		m := Move{From: from, To: to}
		if isCapture {
			m.Flags |= FlagCapture
		}
		*out = append(*out, m)
		return
	}
	// promotions: q r b n
	for _, pt := range []PieceType{Queen, Rook, Bishop, Knight} {
		m := Move{From: from, To: to, Promo: pt, Flags: FlagPromotion}
		if isCapture {
			m.Flags |= FlagCapture
		}
		*out = append(*out, m)
	}
}

func mustSq(s string) Square {
	sq, err := ParseSquare(s)
	if err != nil {
		panic(err)
	}
	return sq
}

func findKing(pos Position, c Color) Square {
	for sq := Square(0); sq < 64; sq++ {
		pc := pos.Board[sq]
		if pc.Type == King && pc.Color == c {
			return sq
		}
	}
	return NoSquare
}

var errIllegalMove = errors.New("illegal move")
