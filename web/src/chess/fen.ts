export type PieceType = "p" | "n" | "b" | "r" | "q" | "k";
export type PieceColor = "w" | "b";

export interface Piece {
  color: PieceColor;
  type: PieceType;
}

export type Board64 = (Piece | null)[]; // index 0 = a1, 63 = h8

function isDigit(ch: string) {
  return ch >= "0" && ch <= "9";
}

export function parseFENBoard(fen: string): Board64 {
  const board: Board64 = new Array(64).fill(null);
  const fields = fen.trim().split(/\s+/);
  const placement = fields[0];
  const ranks = placement.split("/");
  if (ranks.length !== 8) throw new Error("Invalid FEN: ranks");

  for (let fenRank = 0; fenRank < 8; fenRank++) {
    const row = ranks[fenRank];
    let file = 0;
    const rank = 7 - fenRank;
    for (let i = 0; i < row.length; i++) {
      const ch = row[i];
      if (isDigit(ch)) {
        file += Number(ch);
        continue;
      }
      const color = ch === ch.toUpperCase() ? "w" : "b";
      const lower = ch.toLowerCase();
      if (!"pnbrqk".includes(lower)) throw new Error("Invalid FEN: piece char");

      const idx = rank * 8 + file;
      board[idx] = { color: color as any, type: lower as any };
      file += 1;
    }
    if (file !== 8) throw new Error("Invalid FEN: file count");
  }

  return board;
}

export function idxToSquare(idx: number): string {
  const file = idx % 8;
  const rank = Math.floor(idx / 8);
  const fileChar = String.fromCharCode("a".charCodeAt(0) + file);
  return `${fileChar}${rank + 1}`;
}
