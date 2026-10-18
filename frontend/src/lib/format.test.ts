import { describe, expect, it } from "vitest";
import {
  basename,
  breadcrumbs,
  displayType,
  exactBytes,
  extension,
  formatBytes,
  formatDate,
  previewKind,
  progress,
} from "./format";

describe("byte formatting", () => {
  it("uses binary units and preserves exact byte counts separately", () => {
    expect(formatBytes(0)).toBe("0 B");
    expect(formatBytes(1024)).toBe("1 KiB");
    expect(formatBytes(1536)).toBe("1.5 KiB");
    expect(formatBytes(1024 ** 4)).toBe("1 TiB");
    expect(exactBytes(1234567)).toBe("1,234,567 B");
  });
  it("handles invalid sizes and clamps progress", () => {
    expect(formatBytes(Number.NaN)).toBe("0 B");
    expect(formatBytes(-1)).toBe("0 B");
    expect(progress(2, 0)).toBe(0);
    expect(progress(200, 100)).toBe(100);
    expect(progress(-1, 100)).toBe(0);
    expect(progress(25, 100)).toBe(25);
  });
});

describe("object keys", () => {
  it("keeps nested prefixes and filenames intact", () => {
    expect(basename("photos/summer/hello world.jpg")).toBe("hello world.jpg");
    expect(basename("photos/summer/")).toBe("summer");
    expect(extension("folder.with.dot/README")).toBe("");
    expect(extension(".env")).toBe("");
    expect(extension("archive.TAR.GZ")).toBe("gz");
    expect(displayType("photo.JPG")).toBe("JPG");
    expect(displayType("README", "")).toBe("—");
    expect(displayType("photo.jpg", "image/jpeg; charset=binary")).toBe("image/jpeg");
    expect(breadcrumbs("photos/summer/")).toEqual([
      { name: "photos", prefix: "photos/" },
      { name: "summer", prefix: "photos/summer/" },
    ]);
    expect(breadcrumbs("")).toEqual([]);
  });
  it("renders active formats as text and identifies supported media", () => {
    expect(previewKind("icon.svg", "image/svg+xml")).toBe("text");
    expect(previewKind("index.html", "text/html")).toBe("text");
    expect(previewKind("object", "image/png")).toBe("image");
    expect(previewKind("clip.MP4")).toBe("video");
    expect(previewKind("report.pdf")).toBe("pdf");
    expect(previewKind("dump.bin")).toBe("file");
  });
});

describe("dates", () => {
  it("does not show invalid or Go zero timestamps", () => {
    expect(formatDate("")).toBe("—");
    expect(formatDate("invalid")).toBe("—");
    expect(formatDate("0001-01-01T00:00:00Z")).toBe("—");
    expect(formatDate("2026-01-15T12:00:00Z")).toContain("2026");
  });
});
