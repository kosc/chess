package chess

import "fmt"

// ApplyMove applies a move for pos.SideToMove and returns the new position.
// It assumes the move is pseudo-legal; it will validate critical rules (captures, en-passant, castling structure).
func ApplyMove(pos Position, m Move) (Position, error) {
	if !m.IsValid() {
		return Position{}, errIllegalMove
	}
	pc := pos.Board[m.From]
	if pc.IsZero() || pc.Color != pos.SideToMove {
		return Position{}, errIllegalMove
	}

	next := pos // copy

	// reset en passant by default
	next.EnPassant = NoSquare

	// clocks default update (will adjust below)
	next.HalfmoveClock++
	if pc.Type == Pawn || (m.Flags&FlagCapture) != 0 {
		next.HalfmoveClock = 0
	}

	// handle special: castling
	if m.Flags&FlagCastle != 0 {
		return applyCastle(next, m)
	}

	// handle en passant capture
	if m.Flags&FlagEnPassant != 0 {
		if pc.Type != Pawn {
			return Position{}, errIllegalMove
		}
		if pos.EnPassant != m.To {
			return Position{}, errIllegalMove
		}
		// captured pawn is behind target square
		capSq := enPassantCapturedPawnSquare(m.To, pc.Color)
		if capSq == NoSquare {
			return Position{}, errIllegalMove
		}
		cap := next.Board[capSq]
		if cap.Type != Pawn || cap.Color == pc.Color {
			return Position{}, errIllegalMove
		}
		next.Board[capSq] = Piece{}
	}

	// normal capture validation
	dst := next.Board[m.To]
	if dst.IsZero() {
		// ok unless capture-flag set (we allow flag mismatch to keep simple? better validate)
		if m.Flags&FlagCapture != 0 && (m.Flags&FlagEnPassant) == 0 {
			return Position{}, errIllegalMove
		}
	} else {
		if dst.Color == pc.Color {
			return Position{}, errIllegalMove
		}
		// ensure capture flag set (not mandatory, but keep consistent)
	}

	// move piece
	next.Board[m.From] = Piece{}

	moved := pc
	// promotion
	if m.Flags&FlagPromotion != 0 {
		if pc.Type != Pawn {
			return Position{}, errIllegalMove
		}
		if m.Promo != Queen && m.Promo != Rook && m.Promo != Bishop && m.Promo != Knight {
			// default to queen (as per UI fallback)
			m.Promo = Queen
		}
		moved.Type = m.Promo
	}

	next.Board[m.To] = moved

	// set en passant square after double pawn push
	if m.Flags&FlagDoublePawnPush != 0 {
		if pc.Type != Pawn {
			return Position{}, errIllegalMove
		}
		ep := enPassantTargetSquare(m.From, m.To)
		if ep == NoSquare {
			return Position{}, errIllegalMove
		}
		next.EnPassant = ep
	}

	// update castling rights if king/rook moved or rook captured
	next.Castling = updateCastlingRights(next.Castling, pc, m.From, m.To, dst)

	// side to move / fullmove
	next.SideToMove = next.SideToMove.Opp()
	if next.SideToMove == White {
		next.FullmoveNumber++
	}

	return next, nil
}

func applyCastle(pos Position, m Move) (Position, error) {
	pc := pos.Board[m.From]
	if pc.Type != King || pc.Color != pos.SideToMove {
		return Position{}, errIllegalMove
	}

	// validate rights and squares emptiness and rook presence
	// White
	if pc.Color == White && m.From == mustSq("e1") && m.To == mustSq("g1") {
		if pos.Castling&CastleWK == 0 {
			return Position{}, errIllegalMove
		}
		if !pos.Board[mustSq("f1")].IsZero() || !pos.Board[mustSq("g1")].IsZero() {
			return Position{}, errIllegalMove
		}
		rook := pos.Board[mustSq("h1")]
		if rook.Type != Rook || rook.Color != White {
			return Position{}, errIllegalMove
		}
		pos.Board[mustSq("e1")] = Piece{}
		pos.Board[mustSq("h1")] = Piece{}
		pos.Board[mustSq("g1")] = Piece{Type: King, Color: White}
		pos.Board[mustSq("f1")] = Piece{Type: Rook, Color: White}
		pos.Castling &^= (CastleWK | CastleWQ)
		pos.HalfmoveClock++
		pos.EnPassant = NoSquare
		pos.SideToMove = Black
		return pos, nil
	}
	if pc.Color == White && m.From == mustSq("e1") && m.To == mustSq("c1") {
		if pos.Castling&CastleWQ == 0 {
			return Position{}, errIllegalMove
		}
		if !pos.Board[mustSq("d1")].IsZero() || !pos.Board[mustSq("c1")].IsZero() || !pos.Board[mustSq("b1")].IsZero() {
			return Position{}, errIllegalMove
		}
		rook := pos.Board[mustSq("a1")]
		if rook.Type != Rook || rook.Color != White {
			return Position{}, errIllegalMove
		}
		pos.Board[mustSq("e1")] = Piece{}
		pos.Board[mustSq("a1")] = Piece{}
		pos.Board[mustSq("c1")] = Piece{Type: King, Color: White}
		pos.Board[mustSq("d1")] = Piece{Type: Rook, Color: White}
		pos.Castling &^= (CastleWK | CastleWQ)
		pos.HalfmoveClock++
		pos.EnPassant = NoSquare
		pos.SideToMove = Black
		return pos, nil
	}

	// Black
	if pc.Color == Black && m.From == mustSq("e8") && m.To == mustSq("g8") {
		if pos.Castling&CastleBK == 0 {
			return Position{}, errIllegalMove
		}
		if !pos.Board[mustSq("f8")].IsZero() || !pos.Board[mustSq("g8")].IsZero() {
			return Position{}, errIllegalMove
		}
		rook := pos.Board[mustSq("h8")]
		if rook.Type != Rook || rook.Color != Black {
			return Position{}, errIllegalMove
		}
		pos.Board[mustSq("e8")] = Piece{}
		pos.Board[mustSq("h8")] = Piece{}
		pos.Board[mustSq("g8")] = Piece{Type: King, Color: Black}
		pos.Board[mustSq("f8")] = Piece{Type: Rook, Color: Black}
		pos.Castling &^= (CastleBK | CastleBQ)
		pos.HalfmoveClock++
		pos.EnPassant = NoSquare
		pos.SideToMove = White
		pos.FullmoveNumber++
		return pos, nil
	}
	if pc.Color == Black && m.From == mustSq("e8") && m.To == mustSq("c8") {
		if pos.Castling&CastleBQ == 0 {
			return Position{}, errIllegalMove
		}
		if !pos.Board[mustSq("d8")].IsZero() || !pos.Board[mustSq("c8")].IsZero() || !pos.Board[mustSq("b8")].IsZero() {
			return Position{}, errIllegalMove
		}
		rook := pos.Board[mustSq("a8")]
		if rook.Type != Rook || rook.Color != Black {
			return Position{}, errIllegalMove
		}
		pos.Board[mustSq("e8")] = Piece{}
		pos.Board[mustSq("a8")] = Piece{}
		pos.Board[mustSq("c8")] = Piece{Type: King, Color: Black}
		pos.Board[mustSq("d8")] = Piece{Type: Rook, Color: Black}
		pos.Castling &^= (CastleBK | CastleBQ)
		pos.HalfmoveClock++
		pos.EnPassant = NoSquare
		pos.SideToMove = White
		pos.FullmoveNumber++
		return pos, nil
	}

	return Position{}, fmt.Errorf("invalid castle move")
}

func enPassantTargetSquare(from, to Square) Square {
	// from rank -> to rank differs by 2
	if from.File() != to.File() {
		return NoSquare
	}
	if to.Rank()-from.Rank() == 2 {
		return SquareFromFR(from.File(), from.Rank()+1)
	}
	if to.Rank()-from.Rank() == -2 {
		return SquareFromFR(from.File(), from.Rank()-1)
	}
	return NoSquare
}

func enPassantCapturedPawnSquare(target Square, mover Color) Square {
	// If white captures en passant onto rank 6 (0-based 5), the captured pawn is on rank 5 (0-based 4), i.e. one step back.
	// In general: captured pawn is one rank opposite of mover direction.
	if mover == White {
		return SquareFromFR(target.File(), target.Rank()-1)
	}
	return SquareFromFR(target.File(), target.Rank()+1)
}

func updateCastlingRights(cr CastlingRights, moved Piece, from, to Square, captured Piece) CastlingRights {
	// king moves => remove both rights for that color
	if moved.Type == King {
		if moved.Color == White {
			cr &^= (CastleWK | CastleWQ)
		} else {
			cr &^= (CastleBK | CastleBQ)
		}
	}

	// rook moves from initial squares
	if moved.Type == Rook {
		switch from {
		case mustSq("h1"):
			cr &^= CastleWK
		case mustSq("a1"):
			cr &^= CastleWQ
		case mustSq("h8"):
			cr &^= CastleBK
		case mustSq("a8"):
			cr &^= CastleBQ
		}
	}

	// rook captured on initial squares
	if captured.Type == Rook {
		switch to {
		case mustSq("h1"):
			cr &^= CastleWK
		case mustSq("a1"):
			cr &^= CastleWQ
		case mustSq("h8"):
			cr &^= CastleBK
		case mustSq("a8"):
			cr &^= CastleBQ
		}
	}

	return cr
}
