import { useEffect, useRef } from "react";
import type { PlayedMove } from "../api/types";

function notation(uci: string) {
  const promotion = uci.length === 5 ? `=${uci[4].toUpperCase()}` : "";
  return `${uci.slice(0, 2)}–${uci.slice(2, 4)}${promotion}`;
}

export function MoveHistory({ moves }: { moves: PlayedMove[] }) {
  const scrollArea = useRef<HTMLDivElement>(null);
  const rows = new Map<number, { white?: string; black?: string }>();
  for (const move of moves) {
    const row = rows.get(move.number) ?? {};
    row[move.side] = move.uci;
    rows.set(move.number, row);
  }

  useEffect(() => {
    const area = scrollArea.current;
    if (area) area.scrollTop = area.scrollHeight;
  }, [moves.length]);

  return (
    <section className="move-history" aria-labelledby="move-history-title">
      <h2 id="move-history-title">Move history</h2>
      <div className="move-history-scroll" ref={scrollArea}>
        {moves.length === 0 ? <p>No moves yet.</p> : (
          <table>
            <thead><tr><th scope="col">Move</th><th scope="col">White</th><th scope="col">Black</th></tr></thead>
            <tbody>
              {Array.from(rows, ([number, row]) => (
                <tr key={number}>
                  <th scope="row">{number}.</th>
                  <td>{row.white ? notation(row.white) : "—"}</td>
                  <td>{row.black ? notation(row.black) : "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </section>
  );
}
