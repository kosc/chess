import { createGame, getGame, HTTPError } from "./client.ts";
import type { GameState } from "./types.ts";

export const GAME_STORAGE_KEY = "chessweb.gameId";
type GameStorage = Pick<Storage, "getItem" | "setItem">;

let pendingLoad: Promise<GameState> | null = null;

export async function startNewGame(storage: GameStorage = localStorage): Promise<GameState> {
  const game = await createGame({ clockEnabled: false, humanSide: "white" });
  storage.setItem(GAME_STORAGE_KEY, game.id);
  return game;
}

export function loadOrCreateGame(storage: GameStorage = localStorage): Promise<GameState> {
  // React StrictMode may mount the effect twice before the first request completes.
  if (!pendingLoad) {
    pendingLoad = (async () => {
      const id = storage.getItem(GAME_STORAGE_KEY);
      if (id) {
        try {
          return await getGame(id);
        } catch (error: unknown) {
          // Preserve the saved ID on network/server failures so restoration can retry.
          if (!(error instanceof HTTPError) || error.status !== 404) throw error;
        }
      }
      return startNewGame(storage);
    })().finally(() => {
      pendingLoad = null;
    });
  }
  return pendingLoad;
}
