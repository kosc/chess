import type {
  GameState,
  LegalMovesResponse,
  MakeMoveRequest,
  Side,
} from "./types";

const API_BASE = "http://localhost:8080";

async function http<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(API_BASE + path, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {}),
    },
  });

  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    const msg = (data && (data.error as string)) || `HTTP ${res.status}`;
    throw new Error(msg);
  }
  return data as T;
}

export async function createGame(params: {
  clockEnabled: boolean;
  humanSide: Side;
}): Promise<GameState> {
  return http<GameState>("/api/v1/games", {
    method: "POST",
    body: JSON.stringify(params),
  });
}

export async function getGame(id: string): Promise<GameState> {
  return http<GameState>(`/api/v1/games/${id}`, { method: "GET" });
}

export async function legalMoves(
  id: string,
  from: string,
): Promise<LegalMovesResponse> {
  const q = encodeURIComponent(from);
  console.log("legalMoves request", { id, from });
  return http<LegalMovesResponse>(`/api/v1/games/${id}/legal-moves?from=${q}`, {
    method: "POST",
  });
}

export async function makeMove(id: string, move: string): Promise<GameState> {
  const req: MakeMoveRequest = { move };
  return http<GameState>(`/api/v1/games/${id}/move`, {
    method: "POST",
    body: JSON.stringify(req),
  });
}
