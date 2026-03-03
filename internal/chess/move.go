package chess

import (
	"fmt"
	"strings"
)

type MoveFlag uint8

const (
	FlagNone MoveFlag = 0

	FlagCapture MoveFlag = 1 << 0

	FlagDoublePawnPush MoveFlag = 1 << 1
	FlagEnPassant      MoveFlag = 1 << 2
	FlagCastle         MoveFlag = 1 << 3

	FlagPromotion MoveFlag = 1 << 4
)

type Move struct {
	From Square
	To   Square

	Promo PieceType // Queen/Rook/Bishop/Knight if promotion, else NoPiece
	Flags MoveFlag
}

func (m Move) IsValid() bool {
	return m.From.IsValid() && m.To.IsValid() && m.From != m.To
}

func (m Move) UCI() string {
	if !m.IsValid() {
		return ""
	}
	s := m.From.String() + m.To.String()
	if m.Flags&FlagPromotion != 0 {
		switch m.Promo {
		case Queen:
			s += "q"
		case Rook:
			s += "r"
		case Bishop:
			s += "b"
		case Knight:
			s += "n"
		default:
			// fallback for safety
			s += "q"
		}
	}
	return s
}

func ParseUCI(s string) (Move, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if len(s) != 4 && len(s) != 5 {
		return Move{}, fmt.Errorf("invalid UCI length: %q", s)
	}
	from, err := ParseSquare(s[0:2])
	if err != nil {
		return Move{}, err
	}
	to, err := ParseSquare(s[2:4])
	if err != nil {
		return Move{}, err
	}
	m := Move{From: from, To: to, Promo: NoPiece, Flags: FlagNone}
	if len(s) == 5 {
		m.Flags |= FlagPromotion
		switch s[4] {
		case 'q':
			m.Promo = Queen
		case 'r':
			m.Promo = Rook
		case 'b':
			m.Promo = Bishop
		case 'n':
			m.Promo = Knight
		default:
			return Move{}, fmt.Errorf("invalid promotion piece: %q", s)
		}
	}
	return m, nil
}
