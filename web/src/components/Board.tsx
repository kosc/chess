import type { Board64, Piece } from "../chess/fen";
import { idxToSquare } from "../chess/fen";

type Props = {
    board: Board64;
    selected: string | null;
    highlights: Set<string>;
    onSquareClick: (sq: string) => void;
    checkSquare?: string | null;
};

function pieceClass(p: Piece): string {
    return `piece c-${p.color} t-${p.type}`;
}

export function Board({
    board,
    selected,
    highlights,
    onSquareClick,
    checkSquare,
}: Props) {
    const squares: JSX.Element[] = [];

    for (let uiRank = 7; uiRank >= 0; uiRank--) {
        for (let file = 0; file < 8; file++) {
            const idx = uiRank * 8 + file;
            const sq = idxToSquare(idx);
            const isCheck = checkSquare === sq;
            const isLight = (file + uiRank) % 2 === 0;

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
                    type="button"
                >
                    {p ? <span className={pieceClass(p)} /> : null}
                </button>,
            );
        }
    }

    return <div className="board">{squares}</div>;
}
