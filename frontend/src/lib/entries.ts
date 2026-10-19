import type { ListPage, ObjectInfo } from "./api";
import { basename } from "./format";
import { type Kind, kindOf } from "./kinds";

export interface Entry {
  key: string;
  name: string;
  folder: boolean;
  size: number;
  modified: string;
  contentType: string;
  etag: string;
  kind: Kind;
  metadata?: Record<string, string | undefined>;
}

export type SortKey = "name" | "size" | "modified";

export function objectEntry(obj: ObjectInfo, name = basename(obj.key)): Entry {
  return {
    key: obj.key,
    name,
    folder: false,
    size: obj.size,
    modified: obj.lastModified,
    contentType: obj.contentType,
    etag: obj.etag,
    kind: kindOf(obj.key, obj.contentType),
    metadata: obj.metadata,
  };
}

export function folderEntry(key: string): Entry {
  return {
    key,
    name: basename(key),
    folder: true,
    size: 0,
    modified: "",
    contentType: "",
    etag: "",
    kind: "folder",
  };
}

/** Folders come from common prefixes; a marker object for the folder itself is hidden. */
export function entriesFromPages(pages: ListPage[], prefix: string): Entry[] {
  const seen = new Set<string>();
  const out: Entry[] = [];
  for (const page of pages) {
    for (const p of page.prefixes ?? []) {
      if (seen.has(p) || p === prefix) continue;
      seen.add(p);
      out.push(folderEntry(p));
    }
    for (const obj of page.objects ?? []) {
      if (obj.key === prefix || seen.has(obj.key)) continue;
      seen.add(obj.key);
      out.push(obj.key.endsWith("/") ? folderEntry(obj.key) : objectEntry(obj));
    }
  }
  return out;
}

/** Search results show their path below the searched prefix. */
export function entriesFromFind(objects: ObjectInfo[], prefix: string): Entry[] {
  return objects
    .filter((obj) => obj.key !== prefix && !obj.key.endsWith("/"))
    .map((obj) => objectEntry(obj, obj.key.slice(prefix.length) || basename(obj.key)));
}

const collator = new Intl.Collator(undefined, { numeric: true, sensitivity: "base" });

export function sortEntries(entries: Entry[], key: SortKey, dir: 1 | -1): Entry[] {
  return [...entries].sort((a, b) => {
    if (a.folder !== b.folder) return a.folder ? -1 : 1;
    let cmp = 0;
    if (key === "size") cmp = a.size - b.size;
    else if (key === "modified")
      cmp = Date.parse(a.modified || "0") - Date.parse(b.modified || "0");
    if (cmp === 0 || Number.isNaN(cmp)) cmp = collator.compare(a.name, b.name);
    return cmp * dir;
  });
}

export function totalSize(entries: Entry[]): number {
  return entries.reduce((sum, e) => sum + (e.folder ? 0 : e.size), 0);
}
