import { useEffect, useMemo, useRef, useState } from "react";
import type { GameState } from "../api/types";
import { getGame, legalMoves, makeMove, MoveError } from "../api/client";
import { loadOrCreateGame, startNewGame } from "../api/session";
import { parseFENBoard } from "../chess/fen";
import { Board } from "../components/Board";
import { MoveHistory } from "../components/MoveHistory";
import { idxToSquare } from "../chess/fen";

export function GamePage() {
  const [game, setGame] = useState<GameState | null>(null);
  const [selected, setSelected] = useState<string | null>(null);
  const [highlights, setHighlights] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(false);
  const [needsSync, setNeedsSync] = useState(false);
  const [startupAttempt, setStartupAttempt] = useState(0);
  const requestPending = useRef(false);
  const [err, setErr] = useState<string>("");

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        setLoading(true);
        setErr("");
        const g = await loadOrCreateGame();
        if (!cancelled) setGame(g);
      } catch (e: unknown) {
        if (!cancelled) setErr(e instanceof Error ? e.message : String(e));
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, [startupAttempt]);

  const gameID = game?.id;
  const clockRunning = game?.clockEnabled && (game.status === "in_progress" || game.status === "check");

  useEffect(() => {
    if (!gameID || !clockRunning || needsSync) return;
    let cancelled = false;
    const timer = window.setInterval(async () => {
      if (requestPending.current) return;
      requestPending.current = true;
      try {
        const next = await getGame(gameID);
        if (!cancelled) setGame(next);
      } catch (e: unknown) {
        if (!cancelled) setErr(e instanceof Error ? e.message : String(e));
      } finally {
        requestPending.current = false;
      }
    }, 1000);
    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [gameID, clockRunning, needsSync]);

  const board = useMemo(() => {
    if (!game) return null;
    try {
      return parseFENBoard(game.fen);
    } catch {
      return null;
    }
  }, [game]);

  const checkedKingSquare = useMemo(() => {
    if (!game || !board) return null;
    if (game.status !== "check") return null;

    // sideToMove is the side in check
    const kingColor = game.sideToMove === "white" ? "w" : "b";

    for (let i = 0; i < 64; i++) {
      const p = board[i];
      if (p && p.color === kingColor && p.type === "k") {
        return idxToSquare(i);
      }
    }
    return null;
  }, [game, board]);

  async function onSquareClick(sq: string) {
    if (!game || loading || needsSync || requestPending.current || !game.yourTurn) return;
    setErr("");

    // move if destination
    if (selected && highlights.has(sq)) {
      const uci = `${selected}${sq}`;
      requestPending.current = true;
      try {
        setLoading(true);
        const next = await makeMove(game.id, uci);
        setGame(next);
        setSelected(null);
        setHighlights(new Set());
      } catch (e: unknown) {
        if (e instanceof MoveError && e.state) {
          setGame(e.state);
        } else {
          setNeedsSync(true);
        }
        setSelected(null);
        setHighlights(new Set());
        setErr(e instanceof Error ? e.message : String(e));
      } finally {
        requestPending.current = false;
        setLoading(false);
      }
      return;
    }

    // otherwise select
    requestPending.current = true;
    try {
      setLoading(true);
      const res = await legalMoves(game.id, sq);
      setSelected(sq);
      const to = res.toSquares ?? [];
      if (!res.toSquares) {
        console.warn("legal-moves: toSquares is null/undefined", res);
      }
      setHighlights(new Set(to));
    } catch {
      setSelected(null);
      setHighlights(new Set());
    } finally {
      requestPending.current = false;
      setLoading(false);
    }
  }

  async function restoreGame() {
    if (!game || requestPending.current) return;
    requestPending.current = true;
    setLoading(true);
    try {
      const next = await getGame(game.id);
      setGame(next);
      setSelected(null);
      setHighlights(new Set());
      setNeedsSync(false);
      setErr("");
    } catch (e: unknown) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      requestPending.current = false;
      setLoading(false);
    }
  }

  async function newGame() {
    if (requestPending.current) return;
    requestPending.current = true;
    setLoading(true);
    try {
      const next = await startNewGame();
      setGame(next);
      setSelected(null);
      setHighlights(new Set());
      setNeedsSync(false);
      setErr("");
    } catch (e: unknown) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      requestPending.current = false;
      setLoading(false);
    }
  }

  const statusLabels: Record<GameState["status"], string> = {
    in_progress: "Партия идёт", check: "Шах", checkmate: "Мат",
    stalemate: "Пат", draw: "Ничья", timeout: "Время вышло",
  };
  const drawLabels: Record<string, string> = {
    fifty_move: "Правило 50 ходов", threefold: "Троекратное повторение",
    insufficient_material: "Недостаточно материала",
    timeout_insufficient_material: "Недостаточно материала для победы по времени",
  };
  const finished = game && !["in_progress", "check"].includes(game.status);
  const turnLabel = needsSync ? "Нужно восстановить связь" : loading ? "Обновляем позицию…"
    : finished ? "Партия завершена" : game?.yourTurn ? "Ваш ход" : "Ход компьютера";
  const clock = (seconds = 0) => `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, "0")}`;

  return (
    <main className="app-shell">
      <header className="app-header">
        <div className="brand">
          <span className="brand-mark" aria-hidden="true">♞</span>
          <div><p className="eyebrow">Человек и компьютер</p><h1>Шахматы</h1></div>
        </div>
        {game ? <button className="button button-primary" type="button" disabled={loading} onClick={newGame}>
          <span aria-hidden="true">＋</span> Новая партия
        </button> : null}
      </header>

      {err ? <p className="notice notice-error" role="alert">{err}</p> : null}
      {!game && !loading ? (
        <button className="button button-primary" type="button" onClick={() => setStartupAttempt((attempt) => attempt + 1)}>
          Повторить загрузку
        </button>
      ) : null}
      {needsSync ? (
        <div className="notice" role="alert">
          <p>Связь прервалась. Обновите позицию, чтобы продолжить партию.</p>
          <button className="button button-secondary" type="button" disabled={loading} onClick={restoreGame}>Восстановить связь</button>
        </div>
      ) : null}

      {board && game ? (
        <div className="game-layout">
          <section className="board-panel" aria-label="Шахматная доска">
            <div className="player-row">
              <span className="player-avatar avatar-dark" aria-hidden="true">♟</span>
              <div><strong>{game.humanSide === "black" ? "Вы" : "Компьютер"}</strong><span className="player-side">Чёрные фигуры</span></div>
              {game.clockEnabled ? <span className={`clock ${game.sideToMove === "black" && !finished ? "clock-active" : ""}`}>{clock(game.blackSec)}</span> : null}
            </div>
            <Board board={board} selected={selected} highlights={highlights}
              onSquareClick={onSquareClick} checkSquare={checkedKingSquare}
              disabled={loading || needsSync || !game.yourTurn} />
            <div className="player-row">
              <span className="player-avatar avatar-light" aria-hidden="true">♟</span>
              <div><strong>{game.humanSide === "white" ? "Вы" : "Компьютер"}</strong><span className="player-side">Белые фигуры</span></div>
              {game.clockEnabled ? <span className={`clock ${game.sideToMove === "white" && !finished ? "clock-active" : ""}`}>{clock(game.whiteSec)}</span> : null}
            </div>
          </section>
          <aside className="game-sidebar">
            <section className="status-card" aria-labelledby="status-title">
              <p className="eyebrow" id="status-title">Текущая партия</p>
              <div className={`status-badge ${game.status === "check" ? "status-check" : ""}`}>
                <span className="status-dot" aria-hidden="true" />{statusLabels[game.status]}
              </div>
              <h2 aria-live="polite">{turnLabel}</h2>
              <p className="status-description">{game.winner ? `Победили ${game.winner === "white" ? "белые" : "чёрные"}.`
                : game.drawReason ? (drawLabels[game.drawReason] ?? "Партия завершилась вничью.")
                : finished ? "Начните новую партию, чтобы сыграть ещё."
                : "Выберите фигуру — доступные ходы появятся на доске."}</p>
              <div className="game-detail"><span>Последний ход</span><strong>{game.lastMove ? `${game.lastMove.slice(0, 2)}–${game.lastMove.slice(2, 4)}${game.lastMove.length === 5 ? `=${game.lastMove[4].toUpperCase()}` : ""}` : "—"}</strong></div>
            </section>
            <MoveHistory key={game.id} moves={game.moveHistory ?? []} />
          </aside>
        </div>
      ) : <div className="empty-board" role="status"><span aria-hidden="true">♞</span><p>{loading ? "Готовим доску…" : "Не удалось загрузить доску"}</p></div>}
      <footer className="app-footer">Спокойный темп. Продуманные ходы.</footer>
    </main>
  );
}
