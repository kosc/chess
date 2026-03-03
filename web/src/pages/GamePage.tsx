import React, { useEffect, useMemo, useState } from "react";
import type { GameState } from "../api/types";
import { createGame, legalMoves, makeMove } from "../api/client";
import { parseFENBoard } from "../chess/fen";
import { Board } from "../components/Board";
import { idxToSquare } from "../chess/fen";

export function GamePage() {
  const [game, setGame] = useState<GameState | null>(null);
  const [selected, setSelected] = useState<string | null>(null);
  const [highlights, setHighlights] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState<string>("");

  useEffect(() => {
    (async () => {
      try {
        setLoading(true);
        const g = await createGame({ clockEnabled: false, humanSide: "white" });
        setGame(g);
      } catch (e: any) {
        setErr(e?.message ?? String(e));
      } finally {
        setLoading(false);
      }
    })();
  }, []);

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
    if (!game) return;
    setErr("");

    // move if destination
    if (selected && highlights.has(sq)) {
      const uci = `${selected}${sq}`;
      try {
        setLoading(true);
        const next = await makeMove(game.id, uci);
        setGame(next);
        setSelected(null);
        setHighlights(new Set());
      } catch (e: any) {
        setErr(e?.message ?? String(e));
      } finally {
        setLoading(false);
      }
      return;
    }

    // otherwise select
    if (!game.yourTurn) {
      setSelected(null);
      setHighlights(new Set());
      return;
    }

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
      setLoading(false);
    }
  }

  return (
    <div style={{ padding: 16 }}>
      <h1>Chess</h1>

      {err ? <p style={{ color: "crimson" }}>{err}</p> : null}

      {game ? (
        <>
          <p>
            Status: <b>{game.status}</b>{" "}
            {game.drawReason ? <>({game.drawReason})</> : null}
          </p>
          <p>
            You: <b>{game.humanSide}</b>, Side to move: <b>{game.sideToMove}</b>{" "}
            (yourTurn: <b>{String(game.yourTurn)}</b>)
          </p>
          <p>
            Last move: <b>{game.lastMove ?? "-"}</b>
          </p>
          {loading ? <p>Working...</p> : null}
        </>
      ) : (
        <p>{loading ? "Loading..." : "No game"}</p>
      )}

      {board ? (
        <Board
          board={board}
          selected={selected}
          highlights={highlights}
          onSquareClick={onSquareClick}
          checkSquare={checkedKingSquare}
        />
      ) : (
        <p>Board not ready</p>
      )}
    </div>
  );
}
