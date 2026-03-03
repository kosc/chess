package engine

import (
	"context"
	"time"

	"github.com/kosc/chessweb/internal/chess"
)

type Engine struct {
	MaxPly int           // 2..4
	Think  time.Duration // e.g. 800ms
}

func (e Engine) BestMove(pos chess.Position) (chess.Move, bool) {
	maxPly := e.MaxPly
	if maxPly < 2 {
		maxPly = 2
	}
	if maxPly > 4 {
		maxPly = 4
	}
	think := e.Think
	if think <= 0 {
		think = 800 * time.Millisecond
	}
	if think > 5*time.Second {
		think = 5 * time.Second
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(think))
	defer cancel()

	// Iterative deepening 1..maxPly, but you requested 2-4; we'll start at 2.
	var best chess.Move
	bestOk := false
	bestScore := -9999999

	for depth := 2; depth <= maxPly; depth++ {
		m, sc, ok := searchRoot(ctx, pos, depth)
		if ok {
			best, bestScore, bestOk = m, sc, true
		}
		// If time is up, stop and keep last fully-evaluated depth move.
		if ctx.Err() != nil {
			break
		}
		_ = bestScore
	}

	return best, bestOk
}

func searchRoot(ctx context.Context, pos chess.Position, depth int) (chess.Move, int, bool) {
	moves := allLegalMoves(pos)
	if len(moves) == 0 {
		return chess.Move{}, 0, false
	}

	bestScore := -9999999
	var best chess.Move
	ok := false

	for i := range moves {
		if ctx.Err() != nil {
			return chess.Move{}, 0, false
		}
		next, err := chess.ApplyMove(pos, moves[i])
		if err != nil {
			continue
		}
		// Negamax
		sc, completed := negamax(ctx, next, depth-1, -9999999, 9999999)
		if !completed {
			return chess.Move{}, 0, false
		}
		sc = -sc
		if !ok || sc > bestScore {
			bestScore = sc
			best = moves[i]
			ok = true
		}
	}
	return best, bestScore, ok
}

func negamax(ctx context.Context, pos chess.Position, depth int, alpha int, beta int) (score int, completed bool) {
	if ctx.Err() != nil {
		return 0, false
	}

	// Terminal or depth 0
	if depth == 0 {
		return eval(pos), true
	}

	moves := allLegalMoves(pos)
	if len(moves) == 0 {
		// Mate or stalemate
		st := chess.EvaluateStatus(pos, nil)
		switch st.Status {
		case "checkmate":
			// side to move is checkmated => very bad for side to move
			return -100000, true
		case "stalemate":
			return 0, true
		default:
			return 0, true
		}
	}

	for i := range moves {
		if ctx.Err() != nil {
			return 0, false
		}
		next, err := chess.ApplyMove(pos, moves[i])
		if err != nil {
			continue
		}
		sc, ok := negamax(ctx, next, depth-1, -beta, -alpha)
		if !ok {
			return 0, false
		}
		sc = -sc
		if sc > alpha {
			alpha = sc
			if alpha >= beta {
				break
			}
		}
	}
	return alpha, true
}

func allLegalMoves(pos chess.Position) []chess.Move {
	out := make([]chess.Move, 0, 64)
	for sq := chess.Square(0); sq < 64; sq++ {
		pc := pos.Board[sq]
		if pc.IsZero() || pc.Color != pos.SideToMove {
			continue
		}
		out = append(out, chess.LegalMovesFrom(pos, sq)...)
	}
	return out
}

func eval(pos chess.Position) int {
	// Material-only eval from side-to-move perspective (negamax convention)
	// Positive => good for side to move
	score := 0

	for sq := chess.Square(0); sq < 64; sq++ {
		pc := pos.Board[sq]
		if pc.IsZero() {
			continue
		}
		v := pieceValue(pc.Type)
		if pc.Color == pos.SideToMove {
			score += v
		} else {
			score -= v
		}
	}

	return score
}

func pieceValue(t chess.PieceType) int {
	switch t {
	case chess.Pawn:
		return 100
	case chess.Knight:
		return 320
	case chess.Bishop:
		return 330
	case chess.Rook:
		return 500
	case chess.Queen:
		return 900
	case chess.King:
		return 0
	default:
		return 0
	}
}
