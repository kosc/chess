import assert from "node:assert/strict";
import { test } from "node:test";
import { getGame, makeMove, MoveError } from "../src/api/client.ts";
import type { GameState } from "../src/api/types.ts";

const current: GameState = {
  id: "test-game",
  fen: "rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq e6 0 2",
  sideToMove: "white",
  status: "in_progress",
  humanSide: "white",
  yourTurn: true,
  clockEnabled: false,
  lastMove: "e7e5",
};

test("successful move needs only one request", async (t) => {
  const fetchMock = t.mock.method(globalThis, "fetch", async () => Response.json(current));
  assert.deepEqual(await makeMove(current.id, "e2e4"), current);
  assert.equal(fetchMock.mock.callCount(), 1);
});

for (const failure of ["network", "timeout", "truncated_json", "server_error"]) {
  test(`recover server position after ${failure} without repeating the move`, async (t) => {
    const requests: string[] = [];
    t.mock.method(globalThis, "fetch", async (path: string, init: RequestInit) => {
      requests.push(`${init.method} ${path}`);
      if (init.method === "POST") {
        if (failure === "timeout") throw new DOMException("Request timed out", "TimeoutError");
        if (failure === "truncated_json") return new Response('{"id":');
        if (failure === "server_error") return Response.json({ error: "server error" }, { status: 500 });
        throw new TypeError("Failed to fetch");
      }
      return Response.json(current);
    });
    await assert.rejects(makeMove(current.id, "e2e4"), (error: unknown) => {
      assert.ok(error instanceof MoveError);
      assert.deepEqual(error.state, current);
      return true;
    });
    assert.deepEqual(requests, ["POST /api/v1/games/test-game/move", "GET /api/v1/games/test-game"]);
  });
}

test("a rejected move retains its error and supplies the current position", async (t) => {
  t.mock.method(globalThis, "fetch", async (_path: string, init: RequestInit) =>
    init.method === "POST"
      ? Response.json({ error: "illegal move" }, { status: 400 })
      : Response.json(current),
  );
  await assert.rejects(makeMove(current.id, "e2e5"), (error: unknown) => {
    assert.ok(error instanceof MoveError);
    assert.equal(error.message, "illegal move");
    assert.deepEqual(error.state, current);
    return true;
  });
});

test("failed recovery reports unknown state, then an explicit refresh can recover", async (t) => {
  let online = false;
  const methods: string[] = [];
  t.mock.method(globalThis, "fetch", async (_path: string, init: RequestInit) => {
    methods.push(init.method!);
    if (!online) throw new TypeError("offline");
    return Response.json(current);
  });
  await assert.rejects(makeMove(current.id, "e2e4"), (error: unknown) => {
    assert.ok(error instanceof MoveError);
    assert.equal(error.state, null);
    return true;
  });
  online = true;
  assert.deepEqual(await getGame(current.id), current);
  assert.deepEqual(methods, ["POST", "GET", "GET"]);
});

test("invalid JSON during recovery must not unlock the board with an empty state", async (t) => {
  t.mock.method(globalThis, "fetch", async (_path: string, init: RequestInit) => {
    if (init.method === "POST") throw new TypeError("response lost");
    return new Response('{"id":');
  });
  await assert.rejects(makeMove(current.id, "e2e4"), (error: unknown) => {
    assert.ok(error instanceof MoveError);
    assert.equal(error.state, null);
    return true;
  });
});
