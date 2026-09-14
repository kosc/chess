import assert from "node:assert/strict";
import { test } from "node:test";
import { GAME_STORAGE_KEY, loadOrCreateGame, startNewGame } from "../src/api/session.ts";
import type { GameState } from "../src/api/types.ts";

const game: GameState = {
  id: "saved-game",
  fen: "rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq e6 0 2",
  sideToMove: "white",
  status: "in_progress",
  humanSide: "white",
  yourTurn: true,
  clockEnabled: false,
  lastMove: "e7e5",
};

function storageWith(id?: string) {
  const items = new Map<string, string>();
  if (id) items.set(GAME_STORAGE_KEY, id);
  return {
    getItem: (key: string) => items.get(key) ?? null,
    setItem: (key: string, value: string) => { items.set(key, value); },
  };
}

test("first visit creates and saves a game; reloading fetches that game", async (t) => {
  const storage = storageWith();
  const requests: string[] = [];
  t.mock.method(globalThis, "fetch", async (path: string, init: RequestInit) => {
    requests.push(`${init.method} ${path}`);
    return Response.json(game);
  });
  assert.deepEqual(await loadOrCreateGame(storage), game);
  assert.equal(storage.getItem(GAME_STORAGE_KEY), game.id);
  assert.deepEqual(await loadOrCreateGame(storage), game);
  assert.deepEqual(requests, ["POST /api/v1/games", "GET /api/v1/games/saved-game"]);
});

test("completed games are restored instead of replaced", async (t) => {
  const storage = storageWith(game.id);
  const completed = { ...game, status: "checkmate", yourTurn: false };
  const fetchMock = t.mock.method(globalThis, "fetch", async (_path: string, init: RequestInit) => {
    assert.equal(init.method, "GET");
    return Response.json(completed);
  });
  assert.deepEqual(await loadOrCreateGame(storage), completed);
  assert.equal(fetchMock.mock.callCount(), 1);
});

test("404 creates a replacement and saves its ID", async (t) => {
  const storage = storageWith("expired-game");
  const requests: string[] = [];
  t.mock.method(globalThis, "fetch", async (path: string, init: RequestInit) => {
    requests.push(`${init.method} ${path}`);
    return init.method === "GET"
      ? Response.json({ error: "not found" }, { status: 404 })
      : Response.json(game);
  });
  assert.deepEqual(await loadOrCreateGame(storage), game);
  assert.equal(storage.getItem(GAME_STORAGE_KEY), game.id);
  assert.deepEqual(requests, ["GET /api/v1/games/expired-game", "POST /api/v1/games"]);
});

for (const failure of ["network", "server_error", "invalid_json"]) {
  test(`${failure} preserves the saved game and allows retry`, async (t) => {
    const storage = storageWith(game.id);
    let failing = true;
    const methods: string[] = [];
    t.mock.method(globalThis, "fetch", async (_path: string, init: RequestInit) => {
      methods.push(init.method!);
      if (failing) {
        if (failure === "server_error") return Response.json({ error: "unavailable" }, { status: 503 });
        if (failure === "invalid_json") return new Response('{"id":');
        throw new TypeError("offline");
      }
      return Response.json(game);
    });
    await assert.rejects(loadOrCreateGame(storage));
    assert.equal(storage.getItem(GAME_STORAGE_KEY), game.id);
    failing = false;
    assert.deepEqual(await loadOrCreateGame(storage), game);
    assert.deepEqual(methods, ["GET", "GET"]);
  });
}

test("overlapping startup calls share one creation request", async (t) => {
  const storage = storageWith();
  const fetchMock = t.mock.method(globalThis, "fetch", async () => Response.json(game));
  const first = loadOrCreateGame(storage);
  const second = loadOrCreateGame(storage);
  assert.equal(first, second);
  assert.deepEqual(await Promise.all([first, second]), [game, game]);
  assert.equal(fetchMock.mock.callCount(), 1);
  assert.equal(storage.getItem(GAME_STORAGE_KEY), game.id);
});

test("explicit new game replaces the ID only after successful creation", async (t) => {
  const storage = storageWith(game.id);
  let failing = true;
  const next = { ...game, id: "new-game" };
  t.mock.method(globalThis, "fetch", async (_path: string, init: RequestInit) => {
    assert.equal(init.method, "POST");
    if (failing) throw new TypeError("offline");
    return Response.json(next);
  });
  await assert.rejects(startNewGame(storage));
  assert.equal(storage.getItem(GAME_STORAGE_KEY), game.id);
  failing = false;
  assert.deepEqual(await startNewGame(storage), next);
  assert.equal(storage.getItem(GAME_STORAGE_KEY), next.id);
});
