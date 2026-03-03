export type Side = "white" | "black";

export type GameStatus =
  | "in_progress"
  | "check"
  | "checkmate"
  | "stalemate"
  | "draw";

export interface GameState {
  id: string;
  fen: string;
  sideToMove: Side;
  status: GameStatus;
  drawReason?: string;

  clockEnabled: boolean;

  humanSide: Side;
  yourTurn: boolean;

  lastMove?: string;

  whiteSec?: number;
  blackSec?: number;
}

export interface LegalMovesResponse {
  moves: string[];
  toSquares: string[];
}

export interface MakeMoveRequest {
  move: string;
}
