import type { JSX } from "react";
import type { Board64, Piece } from "../chess/fen";
import { idxToSquare } from "../chess/fen";

type Props = {
    board: Board64;
    selected: string | null;
    highlights: Set<string>;
    onSquareClick: (sq: string) => void;
    checkSquare?: string | null;
    disabled?: boolean;
};

function pieceClass(p: Piece): string {
    return `piece c-${p.color}`;
}

const pieceSymbols = {
    p: "♟",
    n: "♞",
    b: "♝",
    r: "♜",
    q: "♛",
    k: "♚",
} satisfies Record<Piece["type"], string>;

const pieceNames = {
    p: "pawn",
    n: "knight",
    b: "bishop",
    r: "rook",
    q: "queen",
    k: "king",
} satisfies Record<Piece["type"], string>;

export function Board({
    board,
    selected,
    highlights,
    onSquareClick,
    checkSquare,
    disabled = false,
}: Props) {
    const squares: JSX.Element[] = [];

    for (let uiRank = 7; uiRank >= 0; uiRank--) {
        for (let file = 0; file < 8; file++) {
            const idx = uiRank * 8 + file;
            const sq = idxToSquare(idx);
            const isCheck = checkSquare === sq;
            const isLight = (file + uiRank) % 2 === 1;

            const p = board[idx];
            const isSelected = selected === sq;
            const isHL = highlights.has(sq);

            squares.push(
                <button
                    key={sq}
                    className={[
                        "square",
                        isLight ? "light" : "dark",
                        isSelected ? "selected" : "",
                        isHL ? "highlight" : "",
                        isCheck ? "in-check" : "",
                    ].join(" ")}
                    onClick={() => onSquareClick(sq)}
                    disabled={disabled}
                    aria-label={p ? `${sq}: ${p.color === "w" ? "white" : "black"} ${pieceNames[p.type]}` : sq}
                    type="button"
                >
                    {p ? <span className={pieceClass(p)} aria-hidden="true">{pieceSymbols[p.type]}</span> : null}
                </button>,
            );
        }
    }

    return (
        <div className="board-frame">
            <div className="board-ranks" aria-hidden="true">
                {[8, 7, 6, 5, 4, 3, 2, 1].map((rank) => <span key={rank}>{rank}</span>)}
            </div>
            <div className="board">{squares}</div>
            <div className="board-files" aria-hidden="true">
                {Array.from("abcdefgh", (file) => <span key={file}>{file}</span>)}
            </div>
        </div>
    );
}
