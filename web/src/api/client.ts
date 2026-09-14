import type {
  GameState,
  LegalMovesResponse,
  MakeMoveRequest,
  Side,
} from "./types";

async function http<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    signal: AbortSignal.timeout(15_000),
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {}),
    },
  });

  if (!res.ok) {
    const data = await res.json().catch(() => ({}));
    const msg = (data && (data.error as string)) || `HTTP ${res.status}`;
    throw new HTTPError(res.status, msg);
  }
  return await res.json() as T;
}

export class HTTPError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "HTTPError";
    this.status = status;
  }
}

export class MoveError extends Error {
  readonly state: GameState | null;

  constructor(cause: unknown, state: GameState | null) {
    super(cause instanceof Error ? cause.message : String(cause));
    this.name = "MoveError";
    this.state = state;
  }
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
  try {
    return await http<GameState>(`/api/v1/games/${id}/move`, {
      method: "POST",
      body: JSON.stringify(req),
    });
  } catch (error: unknown) {
    // The server may have applied the move before its response was lost.
    // Read the current position; never automatically resend the move.
    let state: GameState | null = null;
    try {
      state = await getGame(id);
    } catch {
      // The caller must keep the board locked until a subsequent read succeeds.
    }
    throw new MoveError(error, state);
  }
}
