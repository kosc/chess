export type Side = "white" | "black";

export type GameStatus =
  | "in_progress"
  | "check"
  | "checkmate"
  | "stalemate"
  | "timeout"
  | "draw";

export interface GameState {
  id: string;
  fen: string;
  sideToMove: Side;
  status: GameStatus;
  drawReason?: string;
  winner?: Side;

  clockEnabled: boolean;

  humanSide: Side;
  yourTurn: boolean;

  lastMove?: string;
  moveHistory: PlayedMove[];

  whiteSec?: number;
  blackSec?: number;
}

export interface PlayedMove {
  number: number;
  side: Side;
  uci: string;
}

export interface LegalMovesResponse {
  moves: string[];
  toSquares: string[];
}

export interface MakeMoveRequest {
  move: string;
}
