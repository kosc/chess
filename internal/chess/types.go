package chess

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

type Color uint8

const (
	White Color = iota
	Black
)

func (c Color) String() string {
	if c == White {
		return "white"
	}
	return "black"
}

func (c Color) Opp() Color {
	if c == White {
		return Black
	}
	return White
}

type PieceType uint8

const (
	NoPiece PieceType = iota
	Pawn
	Knight
	Bishop
	Rook
	Queen
	King
)

type Piece struct {
	Type  PieceType
	Color Color
}

func (p Piece) IsZero() bool { return p.Type == NoPiece }

func (p Piece) FENChar() byte {
	// uppercase = white, lowercase = black
	var ch byte
	switch p.Type {
	case Pawn:
		ch = 'p'
	case Knight:
		ch = 'n'
	case Bishop:
		ch = 'b'
	case Rook:
		ch = 'r'
	case Queen:
		ch = 'q'
	case King:
		ch = 'k'
	default:
		return 0
	}
	if p.Color == White {
		ch = byte(unicode.ToUpper(rune(ch)))
	}
	return ch
}

func pieceFromFENChar(ch byte) (Piece, bool) {
	switch ch {
	case 'P':
		return Piece{Type: Pawn, Color: White}, true
	case 'N':
		return Piece{Type: Knight, Color: White}, true
	case 'B':
		return Piece{Type: Bishop, Color: White}, true
	case 'R':
		return Piece{Type: Rook, Color: White}, true
	case 'Q':
		return Piece{Type: Queen, Color: White}, true
	case 'K':
		return Piece{Type: King, Color: White}, true
	case 'p':
		return Piece{Type: Pawn, Color: Black}, true
	case 'n':
		return Piece{Type: Knight, Color: Black}, true
	case 'b':
		return Piece{Type: Bishop, Color: Black}, true
	case 'r':
		return Piece{Type: Rook, Color: Black}, true
	case 'q':
		return Piece{Type: Queen, Color: Black}, true
	case 'k':
		return Piece{Type: King, Color: Black}, true
	default:
		return Piece{}, false
	}
}

type Square uint8

const (
	NoSquare Square = 64
)

func (s Square) IsValid() bool { return s < 64 }

func (s Square) File() int { return int(s % 8) } // 0..7 => a..h
func (s Square) Rank() int { return int(s / 8) } // 0..7 => 1..8

func SquareFromFR(file, rank int) Square {
	// file: 0..7 (a..h), rank: 0..7 (1..8)
	if file < 0 || file > 7 || rank < 0 || rank > 7 {
		return NoSquare
	}
	return Square(rank*8 + file)
}

func ParseSquare(s string) (Square, error) {
	if len(s) != 2 {
		return NoSquare, fmt.Errorf("invalid square %q", s)
	}
	f := s[0]
	r := s[1]
	if f < 'a' || f > 'h' || r < '1' || r > '8' {
		return NoSquare, fmt.Errorf("invalid square %q", s)
	}
	file := int(f - 'a')
	rank := int(r - '1')
	return SquareFromFR(file, rank), nil
}

func (sq Square) String() string {
	if !sq.IsValid() {
		return "-"
	}
	return string([]byte{byte('a' + sq.File()), byte('1' + sq.Rank())})
}

type CastlingRights uint8

const (
	CastleWK CastlingRights = 1 << 0
	CastleWQ CastlingRights = 1 << 1
	CastleBK CastlingRights = 1 << 2
	CastleBQ CastlingRights = 1 << 3
)

type Position struct {
	Board [64]Piece

	SideToMove Color
	Castling   CastlingRights
	EnPassant  Square // target square in algebraic, or NoSquare

	HalfmoveClock  int
	FullmoveNumber int
}

func StartFEN() string {
	return "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
}

func ParseFEN(fen string) (Position, error) {
	fen = strings.TrimSpace(fen)
	if fen == "" {
		fen = StartFEN()
	}

	fields := strings.Fields(fen)
	if len(fields) != 6 {
		return Position{}, fmt.Errorf("FEN must have 6 fields, got %d", len(fields))
	}

	var pos Position
	for i := range pos.Board {
		pos.Board[i] = Piece{}
	}

	// 1) placement
	ranks := strings.Split(fields[0], "/")
	if len(ranks) != 8 {
		return Position{}, errors.New("FEN placement must have 8 ranks")
	}
	// FEN goes rank8 to rank1, our rank index 7..0
	for fenRank := 0; fenRank < 8; fenRank++ {
		row := ranks[fenRank]
		file := 0
		rank := 7 - fenRank
		for i := 0; i < len(row); i++ {
			ch := row[i]
			if ch >= '1' && ch <= '8' {
				file += int(ch - '0')
				continue
			}
			p, ok := pieceFromFENChar(ch)
			if !ok {
				return Position{}, fmt.Errorf("invalid piece char %q in placement", ch)
			}
			if file > 7 {
				return Position{}, errors.New("FEN placement overflow")
			}
			sq := SquareFromFR(file, rank)
			pos.Board[sq] = p
			file++
		}
		if file != 8 {
			return Position{}, errors.New("FEN placement rank must have 8 files")
		}
	}

	// 2) side to move
	switch fields[1] {
	case "w":
		pos.SideToMove = White
	case "b":
		pos.SideToMove = Black
	default:
		return Position{}, fmt.Errorf("invalid side to move %q", fields[1])
	}

	// 3) castling
	pos.Castling = 0
	if fields[2] != "-" {
		for i := 0; i < len(fields[2]); i++ {
			switch fields[2][i] {
			case 'K':
				pos.Castling |= CastleWK
			case 'Q':
				pos.Castling |= CastleWQ
			case 'k':
				pos.Castling |= CastleBK
			case 'q':
				pos.Castling |= CastleBQ
			default:
				return Position{}, fmt.Errorf("invalid castling char %q", fields[2][i])
			}
		}
	}

	// 4) en passant
	if fields[3] == "-" {
		pos.EnPassant = NoSquare
	} else {
		sq, err := ParseSquare(fields[3])
		if err != nil {
			return Position{}, fmt.Errorf("invalid en passant square: %w", err)
		}
		pos.EnPassant = sq
	}

	// 5) halfmove
	var err error
	pos.HalfmoveClock, err = parseInt(fields[4])
	if err != nil || pos.HalfmoveClock < 0 {
		return Position{}, fmt.Errorf("invalid halfmove clock %q", fields[4])
	}

	// 6) fullmove
	pos.FullmoveNumber, err = parseInt(fields[5])
	if err != nil || pos.FullmoveNumber <= 0 {
		return Position{}, fmt.Errorf("invalid fullmove number %q", fields[5])
	}

	// Basic sanity: must have exactly one king each.
	if countKings(pos, White) != 1 || countKings(pos, Black) != 1 {
		return Position{}, errors.New("position must have exactly one white king and one black king")
	}

	return pos, nil
}

func (p Position) FEN() string {
	var sb strings.Builder

	// placement: rank8 to rank1
	for fenRank := 7; fenRank >= 0; fenRank-- {
		empty := 0
		for file := 0; file < 8; file++ {
			sq := SquareFromFR(file, fenRank)
			pc := p.Board[sq]
			if pc.IsZero() {
				empty++
				continue
			}
			if empty > 0 {
				sb.WriteByte(byte('0' + empty))
				empty = 0
			}
			sb.WriteByte(pc.FENChar())
		}
		if empty > 0 {
			sb.WriteByte(byte('0' + empty))
		}
		if fenRank != 0 {
			sb.WriteByte('/')
		}
	}

	// side
	sb.WriteByte(' ')
	if p.SideToMove == White {
		sb.WriteByte('w')
	} else {
		sb.WriteByte('b')
	}

	// castling
	sb.WriteByte(' ')
	if p.Castling == 0 {
		sb.WriteByte('-')
	} else {
		if p.Castling&CastleWK != 0 {
			sb.WriteByte('K')
		}
		if p.Castling&CastleWQ != 0 {
			sb.WriteByte('Q')
		}
		if p.Castling&CastleBK != 0 {
			sb.WriteByte('k')
		}
		if p.Castling&CastleBQ != 0 {
			sb.WriteByte('q')
		}
	}

	// en passant
	sb.WriteByte(' ')
	if p.EnPassant == NoSquare {
		sb.WriteByte('-')
	} else {
		sb.WriteString(p.EnPassant.String())
	}

	// clocks
	sb.WriteByte(' ')
	sb.WriteString(itoa(p.HalfmoveClock))
	sb.WriteByte(' ')
	sb.WriteString(itoa(p.FullmoveNumber))

	return sb.String()
}

func countKings(pos Position, c Color) int {
	n := 0
	for _, pc := range pos.Board {
		if pc.Type == King && pc.Color == c {
			n++
		}
	}
	return n
}

func parseInt(s string) (int, error) {
	if s == "" {
		return 0, errors.New("empty")
	}
	n := 0
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch < '0' || ch > '9' {
			return 0, errors.New("not a number")
		}
		n = n*10 + int(ch-'0')
	}
	return n, nil
}

func itoa(n int) string {
	// small and dependency-free
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoa(-n)
	}
	var buf [32]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + (n % 10))
		n /= 10
	}
	return string(buf[i:])
}
