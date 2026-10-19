import { describe, expect, it } from "vitest";
import { markdownImage, PATH_TEMPLATES, renderTemplate, sanitizeName } from "./template";

const values = {
  date: new Date(2026, 2, 9, 5, 4),
  hash: "abcdef0123456789",
  name: "My Photo",
  ext: "PNG",
};

describe("quick upload paths", () => {
  it("expands every supported placeholder", () => {
    expect(renderTemplate(PATH_TEMPLATES[0], values)).toBe("images/2026/03/abcdef01.png");
    expect(renderTemplate(PATH_TEMPLATES[1], values)).toBe("images/20260309-My-Photo.png");
    expect(renderTemplate(PATH_TEMPLATES[2], values)).toBe("screenshots/My-Photo.png");
    expect(renderTemplate("{yyyy}/{mm}/{dd}/{hh}/{hash}.{ext}", values)).toBe(
      "2026/03/09/05/abcdef01.png",
    );
  });

  it("puts conflict suffixes before the extension", () => {
    expect(renderTemplate("images/{hash}.{ext}", values, "-2")).toBe("images/abcdef01-2.png");
    expect(renderTemplate("images/{hash}", values, "-2")).toBe("images/abcdef01-2");
    expect(renderTemplate("images/{hash}", values, "")).toBe("images/abcdef01");
  });

  it("normalizes unsafe names and empty segments", () => {
    expect(sanitizeName("  quarterly report/final?.pdf")).toBe("quarterly-report-final");
    expect(renderTemplate("//images//{name}.{ext}", values)).toBe("images/My-Photo.png");
    expect(renderTemplate("{name}.{ext}", { ...values, name: "" })).toBe("image.png");
    expect(markdownImage("a[b].png", "https://example.com/a.png")).toBe(
      "![ab.png](https://example.com/a.png)",
    );
  });
});
