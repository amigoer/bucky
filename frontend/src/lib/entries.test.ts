import { describe, expect, it } from "vitest";
import type { ListPage, ObjectInfo } from "./api";
import { entriesFromFind, entriesFromPages, sortEntries } from "./entries";

const page: ListPage = {
  objects: [
    {
      profileId: "p",
      bucket: "b",
      key: "notes.md",
      size: 10,
      etag: "e1",
      lastModified: "2026-01-02T00:00:00Z",
      contentType: "text/markdown",
      metadata: {},
    } as ObjectInfo,
    {
      profileId: "p",
      bucket: "b",
      key: "images/",
      size: 0,
      etag: "",
      lastModified: "",
      contentType: "",
      metadata: {},
    } as ObjectInfo,
  ],
  prefixes: ["images/", "docs/"],
  nextCursor: "",
};

describe("listing entries", () => {
  it("deduplicates prefixes, hides the prefix marker and keeps folders first", () => {
    const entries = entriesFromPages([page], "");
    expect(entries.map((entry) => [entry.key, entry.folder])).toEqual([
      ["images/", true],
      ["docs/", true],
      ["notes.md", false],
    ]);
    const sorted = sortEntries(entries, "name", 1);
    expect(sorted.map((entry) => entry.name)).toEqual(["docs", "images", "notes.md"]);
    expect(sortEntries(entries, "name", -1).map((entry) => entry.name)).toEqual([
      "images",
      "docs",
      "notes.md",
    ]);
  });

  it("hides marker objects that match the current prefix", () => {
    const entries = entriesFromPages([page], "images/");
    expect(entries.map((entry) => entry.key)).toEqual(["docs/", "notes.md"]);
    expect(
      entriesFromPages([{ ...page, prefixes: ["images/"] }], "images/").map((e) => e.key),
    ).toEqual(["notes.md"]);
  });

  it("keeps paths relative to the search scope", () => {
    const objects = [
      { ...page.objects[0], key: "images/2026/hero.png" },
      { ...page.objects[0], key: "images/folder/" },
      { ...page.objects[0], key: "images/" },
    ] as ObjectInfo[];
    const entries = entriesFromFind(objects, "images/");
    expect(entries.map((entry) => [entry.key, entry.name])).toEqual([
      ["images/2026/hero.png", "2026/hero.png"],
    ]);
  });
});
