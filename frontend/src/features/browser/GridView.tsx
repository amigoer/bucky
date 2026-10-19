import { useVirtualizer } from "@tanstack/react-virtual";
import { useEffect, useRef, useState } from "react";
import { KIND_ICON } from "@/components/file-tile";
import { tileStyle } from "@/lib/kinds";
import { cn } from "@/lib/utils";
import { EntryContextMenu } from "./EntryContextMenu";
import type { EntryViewProps } from "./types";

const TILE_W = 112;
const TILE_H = 120;
const GAP = 8;
const PAD = 16;

export function GridView({
  entries,
  selection,
  focusKey,
  onSelect,
  onOpen,
  onLoadMore,
  location,
  readOnly,
}: EntryViewProps) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const [width, setWidth] = useState(0);

  useEffect(() => {
    const element = scrollRef.current;
    if (!element) return;
    const observer = new ResizeObserver((observed) => {
      setWidth(observed[0]?.contentRect.width ?? element.clientWidth);
    });
    observer.observe(element);
    setWidth(element.clientWidth);
    return () => observer.disconnect();
  }, []);

  const columns = Math.max(1, Math.floor((width - PAD * 2 + GAP) / (TILE_W + GAP)));
  const rows = Math.ceil(entries.length / columns);

  const virtualizer = useVirtualizer({
    count: rows,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => TILE_H + GAP,
    overscan: 5,
  });

  const virtualItems = virtualizer.getVirtualItems();
  const lastVisible = virtualItems.at(-1)?.index ?? 0;
  useEffect(() => {
    if (rows > 0 && lastVisible >= rows - 3) onLoadMore?.();
  }, [lastVisible, rows, onLoadMore]);

  const focusIndex = focusKey ? entries.findIndex((entry) => entry.key === focusKey) : -1;
  useEffect(() => {
    if (focusIndex >= 0) {
      virtualizer.scrollToIndex(Math.floor(focusIndex / columns), { align: "auto" });
    }
  }, [focusIndex, columns, virtualizer]);

  return (
    <div
      ref={scrollRef}
      className="min-h-0 flex-1 overflow-y-auto"
      role="listbox"
      aria-multiselectable
    >
      <div className="relative" style={{ height: virtualizer.getTotalSize() + PAD }}>
        {virtualItems.map((row) => {
          const start = row.index * columns;
          return (
            <div
              key={row.key}
              className="absolute flex"
              style={{
                transform: `translateY(${row.start + PAD}px)`,
                left: PAD,
                gap: GAP,
              }}
            >
              {entries.slice(start, start + columns).map((entry) => {
                const selected = selection.has(entry.key);
                const Icon = KIND_ICON[entry.kind];
                const style =
                  entry.kind === "image"
                    ? {
                        background:
                          "repeating-linear-gradient(135deg, var(--stripe) 0 5px, transparent 5px 10px), var(--tile-image)",
                        color: "var(--kind-image-fg)",
                      }
                    : tileStyle(entry.kind);
                return (
                  <EntryContextMenu
                    key={entry.key}
                    entry={entry}
                    location={location}
                    readOnly={readOnly}
                  >
                    <div
                      role="option"
                      aria-selected={selected}
                      tabIndex={-1}
                      onKeyDown={(event) => {
                        if (event.key === "Enter" || event.key === " ") {
                          event.preventDefault();
                          onSelect(entry, {
                            metaKey: event.metaKey,
                            ctrlKey: event.ctrlKey,
                            shiftKey: event.shiftKey,
                          });
                        }
                      }}
                      data-entry-key={entry.key}
                      title={entry.name}
                      onClick={(event) =>
                        onSelect(entry, {
                          metaKey: event.metaKey,
                          ctrlKey: event.ctrlKey,
                          shiftKey: event.shiftKey,
                        })
                      }
                      onDoubleClick={() => onOpen(entry)}
                      className={cn(
                        "flex flex-none flex-col items-center gap-1.5 rounded-xl p-2 transition-[background-color,box-shadow]",
                        selected
                          ? "bg-primary-tint shadow-[0_0_0_2.5px_var(--primary),0_0_0_6px_var(--primary-soft)]"
                          : "hover:bg-hover active:bg-muted",
                      )}
                      style={{ width: TILE_W, height: TILE_H }}
                    >
                      <span
                        className="grid h-[72px] w-24 flex-none place-items-center rounded-lg transition-transform"
                        style={style}
                      >
                        <Icon className="size-8" strokeWidth={1.7} />
                      </span>
                      <span
                        className={cn(
                          "line-clamp-2 w-full text-center text-[11.5px] leading-[1.25] break-all",
                          selected ? "font-semibold text-primary-text" : "text-foreground",
                        )}
                      >
                        {entry.name}
                      </span>
                    </div>
                  </EntryContextMenu>
                );
              })}
            </div>
          );
        })}
      </div>
    </div>
  );
}
