import type { Entry } from "./tool/entry.ts";
import { useMemo, useState } from "react";
import { formatBytes } from "./tool/utils.ts";

const PAGE_SIZE = 50;

export function ItemBrowser({ entry, onSelect }: { entry: Entry; onSelect: (entry: Entry) => void }) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(0);
  const children = entry.getChildren();
  const filtered = useMemo(() => open
    ? children.filter(child => child.getName().toLocaleLowerCase().includes(query.toLocaleLowerCase()))
        .sort((a, b) => b.getSize() - a.getSize())
    : [], [children, query, open]);

  return (
    <aside className="item-browser" aria-label="Browse items">
      <button type="button" aria-expanded={open} onClick={() => setOpen(!open)}>
        {open
          ? "Close item list"
          : `Browse all ${children.length.toLocaleString()} items`}
      </button>
      {open && (
        <>
          <p>Small areas appear as you zoom. This list includes every direct item.</p>
          <input
            type="search"
            aria-label="Filter items"
            placeholder="Filter by name"
            value={query}
            onChange={(event) => {
              setQuery(event.target.value);
              setPage(0);
            }}
          />
          <ul>
            {filtered.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE).map(child => (
              <li key={child.getID()}>
                <button type="button" aria-label={`${child.getName()} ${formatBytes(child.getSize())}`} onClick={() => onSelect(child)}>
                  <span>{child.getName()}</span>
                  <span>{formatBytes(child.getSize())}</span>
                </button>
              </li>
            ))}
          </ul>
          <div className="item-pages">
            <button type="button" disabled={page === 0} onClick={() => setPage(page - 1)}>Previous</button>
            <span>
              {filtered.length === 0
                ? "No matching items"
                : `${page * PAGE_SIZE + 1}–${Math.min((page + 1) * PAGE_SIZE, filtered.length)} of ${filtered.length}`}
            </span>
            <button type="button" disabled={(page + 1) * PAGE_SIZE >= filtered.length} onClick={() => setPage(page + 1)}>Next</button>
          </div>
        </>
      )}
    </aside>
  );
}
