import { describe, expect, it } from "vitest";
import type { ItemResult, OperationPlan } from "./api";
import { isFinished, overwriteCount, resultCounts, resultState, secondsLeft } from "./plan";

const result = (status: string, code?: string) =>
  ({ source: {}, status, error: code ? { code } : undefined }) as ItemResult;

describe("plan results", () => {
  it("maps statuses and error codes to display states", () => {
    expect(resultState(result("completed"))).toBe("ok");
    expect(resultState(result("queued"))).toBe("ok");
    expect(resultState(result("task:abc"))).toBe("ok");
    expect(resultState(result("failed", "conflict"))).toBe("conflict");
    expect(resultState(result("failed", "not_found"))).toBe("skip");
    expect(resultState(result("failed", "forbidden"))).toBe("fail");
    expect(resultState(result("running"))).toBe("pending");
  });

  it("counts only finished results", () => {
    const plan = {
      results: [
        result("completed"),
        result("task:abc"),
        result("failed", "not_found"),
        result("failed", "conflict"),
        result("failed", "forbidden"),
        result("running"),
      ],
    } as OperationPlan;
    expect(resultCounts(plan)).toEqual({ ok: 2, fail: 1, skip: 1, conflict: 1 });
  });
});

describe("plan timing and items", () => {
  const now = Date.parse("2026-10-18T12:00:00Z");
  it("never reports negative time left", () => {
    expect(secondsLeft({ expiresAt: "2026-10-18T12:00:30Z" }, now)).toBe(30);
    expect(secondsLeft({ expiresAt: "2026-10-18T11:59:00Z" }, now)).toBe(0);
    expect(secondsLeft({ expiresAt: "not a date" }, now)).toBe(0);
  });

  it("counts overwrites and terminal statuses", () => {
    const plan = {
      items: [{ existing: null }, { existing: { key: "a" } }, { existing: { key: "b" } }],
      status: "partial",
    } as unknown as OperationPlan;
    expect(overwriteCount(plan)).toBe(2);
    expect(isFinished(plan)).toBe(true);
    expect(isFinished({ ...plan, status: "pending" })).toBe(false);
    expect(isFinished({ ...plan, status: "running" })).toBe(false);
    expect(isFinished({ ...plan, status: "interrupted" })).toBe(true);
  });
});
