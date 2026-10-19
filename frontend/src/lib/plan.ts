import type { ItemResult, OperationPlan } from "./api";

export type ResultState = "ok" | "fail" | "skip" | "conflict" | "pending";

export function resultState(result: ItemResult): ResultState {
  if (result.status === "completed" || result.status === "queued") return "ok";
  if (result.status.startsWith("task:")) return "ok";
  const code = result.error?.code ?? "";
  if (code === "conflict") return "conflict";
  if (code === "not_found") return "skip";
  if (result.status === "failed") return "fail";
  return "pending";
}

export function resultCounts(plan: OperationPlan) {
  const counts = { ok: 0, fail: 0, skip: 0, conflict: 0 };
  for (const r of plan.results ?? []) {
    const state = resultState(r);
    if (state !== "pending") counts[state] += 1;
  }
  return counts;
}

export function secondsLeft(plan: Pick<OperationPlan, "expiresAt">, now = Date.now()): number {
  const expires = Date.parse(plan.expiresAt);
  if (!Number.isFinite(expires)) return 0;
  return Math.max(0, Math.floor((expires - now) / 1000));
}

export function overwriteCount(plan: OperationPlan): number {
  return (plan.items ?? []).filter((item) => item.existing).length;
}

export function isFinished(plan: OperationPlan): boolean {
  return plan.status === "completed" || plan.status === "partial" || plan.status === "interrupted";
}
