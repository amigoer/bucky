import { describe, expect, it } from "vitest";
import type { CloudflareSignIn } from "./api";
import {
  initialSignInState,
  isLoopbackRedirect,
  normalizeScopes,
  type SignInState,
  signInReducer,
} from "./cloudflare";

const session = (over: Record<string, unknown> = {}) =>
  ({
    sessionId: "s1",
    state: "waiting",
    authUrl: "https://dash.cloudflare.com/oauth2/auth?state=abc",
    redirectUri: "http://127.0.0.1:8976/oauth/callback",
    accounts: [],
    ...over,
  }) as unknown as CloudflareSignIn;

describe("isLoopbackRedirect", () => {
  it("accepts loopback HTTP callbacks with a port", () => {
    expect(isLoopbackRedirect("http://127.0.0.1:8976/oauth/callback")).toBe(true);
    expect(isLoopbackRedirect("http://localhost:1234/cb")).toBe(true);
    expect(isLoopbackRedirect(" http://127.0.0.1:8976/oauth/callback ")).toBe(true);
  });

  it("rejects anything else", () => {
    for (const value of [
      "https://127.0.0.1:8976/oauth/callback",
      "http://127.0.0.1/oauth/callback",
      "http://example.com:8976/oauth/callback",
      "http://127.0.0.1:8976/cb?x=1",
      "http://127.0.0.1:8976/cb#frag",
      "http://user@127.0.0.1:8976/cb",
      "bucky://oauth/callback",
      "",
    ]) {
      expect(isLoopbackRedirect(value), value).toBe(false);
    }
  });
});

describe("normalizeScopes", () => {
  it("trims, lowercases, drops duplicates and invalid names", () => {
    expect(normalizeScopes([" Workers-R2.Read ", "workers-r2.read", "not a scope", ""])).toEqual([
      "workers-r2.read",
    ]);
  });
});

describe("signInReducer", () => {
  const started = signInReducer(initialSignInState, { type: "started", session: session() });

  it("waits after starting and remembers the session", () => {
    expect(started.phase).toBe("waiting");
    expect(started.sessionId).toBe("s1");
    expect(started.authUrl).toContain("/auth");
  });

  it("moves to ready with the authorized accounts and preselects one", () => {
    const ready = signInReducer(started, {
      type: "updated",
      session: session({
        state: "authorized",
        accounts: [
          { id: "a1", name: "First" },
          { id: "a2", name: "Second" },
        ],
      }),
    });
    expect(ready.phase).toBe("ready");
    expect(ready.accounts).toHaveLength(2);
    expect(ready.accountId).toBe("a1");
  });

  it("reports a failed sign-in with the provider message", () => {
    const failed = signInReducer(started, {
      type: "updated",
      session: session({
        state: "failed",
        error: { code: "permission_denied", message: "Cloudflare authorization was refused" },
      }),
    });
    expect(failed.phase).toBe("error");
    expect(failed.error).toBe("Cloudflare authorization was refused");
  });

  it("ignores updates from another session", () => {
    const other = session({ sessionId: "s2", state: "authorized" });
    expect(signInReducer(started, { type: "updated", session: other })).toBe(started);
  });

  it("keeps the chosen account when it is still authorized", () => {
    const ready: SignInState = { ...started, phase: "ready", accounts: [{ id: "a1", name: "A" }] };
    const picked = signInReducer(ready, { type: "account", accountId: "a2" });
    expect(picked.accountId).toBe("a2");
    const again = signInReducer(picked, {
      type: "updated",
      session: session({ state: "authorized", accounts: [{ id: "a1", name: "A" }] }),
    });
    expect(again.accountId).toBe("a2");
  });
});
