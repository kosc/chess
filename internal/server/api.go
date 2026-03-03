package server

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/kosc/chessweb/internal/chess"
	"github.com/kosc/chessweb/internal/engine"
)

type createGameRequest struct {
	// Если пусто — стартовая позиция.
	FEN string `json:"fen,omitempty"`

	// "white" | "black" — кем играет человек. По умолчанию "white".
	HumanSide string `json:"humanSide,omitempty"`

	// Включить/выключить таймер.
	ClockEnabled bool `json:"clockEnabled"`

	// Секунды на игрока, если ClockEnabled=true. По умолчанию 600.
	InitialSeconds int `json:"initialSeconds,omitempty"`

	// Сложность/глубина бота: 2..4 (ply). По умолчанию 3.
	BotMaxPly int `json:"botMaxPly,omitempty"`

	// Лимит времени на ход бота (ms). По умолчанию 800.
	BotMoveTimeMs int `json:"botMoveTimeMs,omitempty"`
}

type gameStateResponse struct {
	ID           string `json:"id"`
	FEN          string `json:"fen"`
	SideToMove   string `json:"sideToMove"`
	Status       string `json:"status"`
	DrawReason   string `json:"drawReason,omitempty"`
	ClockEnabled bool   `json:"clockEnabled"`

	// add:
	HumanSide string `json:"humanSide"`
	YourTurn  bool   `json:"yourTurn"`

	LastMove string `json:"lastMove,omitempty"`

	WhiteSec int `json:"whiteSec,omitempty"`
	BlackSec int `json:"blackSec,omitempty"`
}

type makeMoveRequest struct {
	// UCI ход: e2e4, e7e8q и т.п.
	Move string `json:"move"`
}

type legalMovesResponse struct {
	Moves     []string `json:"moves"`
	ToSquares []string `json:"toSquares"`
}

// Временное in-memory хранилище (в v1 достаточно).
// Позже можно заменить на Redis/DB, и добавить WebSocket.
type gameStore struct {
	mu    sync.RWMutex
	games map[string]*game
}

var store = &gameStore{games: map[string]*game{}}

type game struct {
	id string

	// TODO: сюда добавим реальную позицию (internal/chess.Position)
	fen string

	humanSide   string
	status      string
	drawReason  string
	sideToMove  string
	lastMoveUCI string

	clockEnabled bool
	whiteSec     int
	blackSec     int
	// чей ход начался (для “шахматных часов”)
	turnStartedAt time.Time

	botMaxPly     int
	botMoveTimeMs int

	historyKeys []string // repetition keys
}

func chessKey(pos chess.Position) string {
	// same logic as chess.repetitionKey, but we keep server-side.
	f := pos.FEN()
	fields := strings.Fields(f)
	// placement stm castling ep
	return fields[0] + " " + fields[1] + " " + fields[2] + " " + fields[3]
}

func (g *game) spendTurnTime() {
	if !g.clockEnabled {
		return
	}
	now := time.Now()
	elapsed := int(now.Sub(g.turnStartedAt).Seconds())
	if elapsed < 0 {
		elapsed = 0
	}
	if g.sideToMove == "white" {
		g.whiteSec -= elapsed
		if g.whiteSec < 0 {
			g.whiteSec = 0
		}
	} else {
		g.blackSec -= elapsed
		if g.blackSec < 0 {
			g.blackSec = 0
		}
	}
	g.turnStartedAt = now
}

func (g *game) flagFallen() (bool, string) {
	if !g.clockEnabled {
		return false, ""
	}
	if g.whiteSec <= 0 {
		return true, "white"
	}
	if g.blackSec <= 0 {
		return true, "black"
	}
	return false, ""
}

func handleCreateGame(w http.ResponseWriter, r *http.Request) {
	var req createGameRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	normalizeCreateReq(&req)
	pos, err := chess.ParseFEN(req.FEN)
	hist := []string{chessKey(pos)}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid FEN: " + err.Error()})
		return
	}
	normFEN := pos.FEN()

	g := &game{
		id:            newID(),
		fen:           normFEN,
		humanSide:     req.HumanSide,
		status:        "in_progress",
		sideToMove:    pos.SideToMove.String(),
		clockEnabled:  req.ClockEnabled,
		whiteSec:      req.InitialSeconds,
		blackSec:      req.InitialSeconds,
		turnStartedAt: time.Now(),
		botMaxPly:     req.BotMaxPly,
		botMoveTimeMs: req.BotMoveTimeMs,
		historyKeys:   hist,
	}

	// If it's bot to move at start (e.g. human plays black), make an opening move immediately.
	if g.sideToMove != g.humanSide {
		curPos, err := chess.ParseFEN(g.fen)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "corrupt initial state: " + err.Error()})
			return
		}

		botMoveUCI, ok := pickFirstLegalMove(curPos)
		if ok {
			bm, _ := chess.ParseUCI(botMoveUCI)
			legal := chess.LegalMovesFrom(curPos, bm.From)
			var chosen *chess.Move
			for i := range legal {
				if legal[i].UCI() == normalizePromoUCI(botMoveUCI) {
					chosen = &legal[i]
					break
				}
			}
			if chosen != nil {
				next, err := chess.ApplyMove(curPos, *chosen)
				if err == nil {
					g.lastMoveUCI = chosen.UCI()
					g.fen = next.FEN()
					g.sideToMove = next.SideToMove.String()
					g.historyKeys = append(g.historyKeys, chessKey(next))

					st := chess.EvaluateStatus(next, g.historyKeys)
					g.status = st.Status
					g.drawReason = st.DrawReason
					g.turnStartedAt = time.Now()
				}
			}
		}
	}
	// TODO: если req.FEN пусто — поставить стартовый FEN
	// TODO: распарсить FEN в internal/chess.Position, валидировать, выставить sideToMove

	store.mu.Lock()
	store.games[g.id] = g
	store.mu.Unlock()

	writeJSON(w, http.StatusCreated, toGameStateResponse(g))
}

func handleGetGame(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}

	g := storeGet(id)
	if g == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}

	writeJSON(w, http.StatusOK, toGameStateResponse(g))
}

func handleLegalMoves(w http.ResponseWriter, r *http.Request, id string) {
	g := storeGet(id)
	if g == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}

	fromStr := strings.TrimSpace(r.URL.Query().Get("from"))
	if fromStr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "missing query param: from"})
		return
	}

	fromSq, err := chess.ParseSquare(fromStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid from square: " + err.Error()})
		return
	}

	pos, err := chess.ParseFEN(g.fen)
	if err != nil {
		// это уже ошибка состояния сервера (FEN должен быть валиден)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "corrupt game state: " + err.Error()})
		return
	}

	moves := chess.LegalMovesFrom(pos, fromSq)

	out := make([]string, 0, len(moves))
	to := make([]string, 0, len(moves))
	seen := make(map[string]struct{}, len(moves))

	for _, m := range moves {
		uci := m.UCI()
		out = append(out, uci)

		ts := m.To.String()
		if _, ok := seen[ts]; !ok {
			seen[ts] = struct{}{}
			to = append(to, ts)
		}
	}

	writeJSON(w, http.StatusOK, legalMovesResponse{
		Moves:     out,
		ToSquares: to,
	})
}

func handleMakeMove(w http.ResponseWriter, r *http.Request, id string) {
	g := storeGet(id)
	if g == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}

	if g.status != "in_progress" && g.status != "check" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "game is finished"})
		return
	}

	var req makeMoveRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	req.Move = strings.TrimSpace(req.Move)
	if req.Move == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "move is required"})
		return
	}

	// Spend time for the player who is about to move (current sideToMove in g)
	g.spendTurnTime()
	if fallen, side := g.flagFallen(); fallen {
		// time loss
		g.status = "draw" // for simplicity? In chess it's loss on time unless insufficient mating material.
		// We'll implement proper result later; for now mark finished.
		g.drawReason = "time_" + side
		writeJSON(w, http.StatusOK, toGameStateResponse(g))
		return
	}

	pos, err := chess.ParseFEN(g.fen)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "corrupt game state: " + err.Error()})
		return
	}

	uciMove, err := chess.ParseUCI(req.Move)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid UCI: " + err.Error()})
		return
	}

	legal := chess.LegalMovesFrom(pos, uciMove.From)
	chosen := matchLegalMove(legal, req.Move)
	if chosen == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "illegal move"})
		return
	}

	next, err := chess.ApplyMove(pos, *chosen)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "illegal move"})
		return
	}

	// Update game state after human move
	g.lastMoveUCI = chosen.UCI()
	g.fen = next.FEN()
	g.sideToMove = next.SideToMove.String()
	g.historyKeys = append(g.historyKeys, chessKey(next))

	st := chess.EvaluateStatus(next, g.historyKeys)
	g.status = st.Status
	g.drawReason = st.DrawReason
	g.turnStartedAt = time.Now()

	// If game ended after human move — return
	if g.status != "in_progress" && g.status != "check" {
		writeJSON(w, http.StatusOK, toGameStateResponse(g))
		return
	}

	if g.sideToMove != g.humanSide {
		eng := engine.Engine{
			MaxPly: g.botMaxPly,
			Think:  time.Duration(g.botMoveTimeMs) * time.Millisecond,
		}
		bm, ok := eng.BestMove(next)
		if !ok {
			// no legal moves
		} else {
			botMoveUCI := bm.UCI()
			// дальше как у тебя: fromSq из botMoveUCI[0:2], legal2, matchLegalMove, ApplyMove...
			// Spend clock time for bot side (its turn is running)
			g.spendTurnTime()
			if fallen, side := g.flagFallen(); fallen {
				g.status = "draw"
				g.drawReason = "time_" + side
				writeJSON(w, http.StatusOK, toGameStateResponse(g))
				return
			}

			// Determine "from" square from botMoveUCI
			if len(botMoveUCI) < 4 {
				// should never happen
				writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "bot produced invalid move"})
				return
			}

			fromSq, err := chess.ParseSquare(botMoveUCI[0:2])
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "bot produced invalid move"})
				return
			}

			legal2 := chess.LegalMovesFrom(next, fromSq)
			chosen2 := matchLegalMove(legal2, botMoveUCI)
			if chosen2 != nil {
				next2, err := chess.ApplyMove(next, *chosen2)
				if err == nil {
					g.lastMoveUCI = chosen2.UCI()
					g.fen = next2.FEN()
					g.sideToMove = next2.SideToMove.String()
					g.historyKeys = append(g.historyKeys, chessKey(next2))

					st2 := chess.EvaluateStatus(next2, g.historyKeys)
					g.status = st2.Status
					g.drawReason = st2.DrawReason
					g.turnStartedAt = time.Now()
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, toGameStateResponse(g))
}

func normalizePromoUCI(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	// if client sends e7e8 (no piece), we treat as queen by default
	if len(s) == 4 {
		// we can't know if it's promo without context, but comparing with legal moves
		// we will accept by expanding in comparison if needed.
		return s
	}
	return s
}

func matchLegalMove(legal []chess.Move, uci string) *chess.Move {
	uci = strings.TrimSpace(strings.ToLower(uci))

	// 1) exact match
	for i := range legal {
		if legal[i].UCI() == uci {
			return &legal[i]
		}
	}

	// 2) if uci is 4 chars, allow matching promotion by assuming queen
	if len(uci) == 4 {
		uciQ := uci + "q"
		for i := range legal {
			if legal[i].UCI() == uciQ {
				return &legal[i]
			}
		}
	}

	return nil
}

func pickFirstLegalMove(pos chess.Position) (string, bool) {
	var fallback string
	for sq := chess.Square(0); sq < 64; sq++ {
		pc := pos.Board[sq]
		if pc.IsZero() || pc.Color != pos.SideToMove {
			continue
		}
		ms := chess.LegalMovesFrom(pos, sq)
		for _, m := range ms {
			if m.Flags&chess.FlagCapture != 0 {
				return m.UCI(), true
			}
			if fallback == "" {
				fallback = m.UCI()
			}
		}
	}
	if fallback != "" {
		return fallback, true
	}
	return "", false
}

func handleLegalMovesRoute(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	handleLegalMoves(w, r, id)
}

func handleMakeMoveRoute(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	handleMakeMove(w, r, id)
}

func toGameStateResponse(g *game) gameStateResponse {
	return gameStateResponse{
		ID:           g.id,
		FEN:          g.fen,
		SideToMove:   g.sideToMove,
		Status:       g.status,
		DrawReason:   g.drawReason,
		ClockEnabled: g.clockEnabled,

		HumanSide: g.humanSide,
		YourTurn:  g.sideToMove == g.humanSide,

		LastMove: g.lastMoveUCI,

		WhiteSec: g.whiteSec,
		BlackSec: g.blackSec,
	}
}

func storeGet(id string) *game {
	store.mu.RLock()
	defer store.mu.RUnlock()
	return store.games[id]
}

func normalizeCreateReq(req *createGameRequest) {
	req.HumanSide = strings.ToLower(strings.TrimSpace(req.HumanSide))
	if req.HumanSide != "black" {
		req.HumanSide = "white"
	}
	if req.InitialSeconds <= 0 {
		req.InitialSeconds = 10 * 60
	}
	if req.BotMaxPly < 2 {
		req.BotMaxPly = 3
	}
	if req.BotMaxPly > 4 {
		req.BotMaxPly = 4
	}
	if req.BotMoveTimeMs <= 0 {
		req.BotMoveTimeMs = 800
	}
	if req.BotMoveTimeMs > 5000 {
		req.BotMoveTimeMs = 5000
	}
	// FEN: пустой разрешён (значит стартовая позиция)
	req.FEN = strings.TrimSpace(req.FEN)
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
