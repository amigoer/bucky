import { describe, expect, it, vi } from "vitest";
import type { CloudflareSignIn } from "./api";
import {
  includesCloudflareAccount,
  initialSignInState,
  isLoopbackRedirect,
  normalizeScopes,
  SignInSession,
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

describe("SignInSession", () => {
  const backend = () => ({
    StartCloudflareSignIn: vi.fn(async () => session()),
    CancelCloudflareSignIn: vi.fn(async (_id: string) => {}),
  });

  it("cancels the previous attempt before restarting", async () => {
    const api = backend();
    const flow = new SignInSession(api);
    await flow.start();
    api.StartCloudflareSignIn.mockImplementationOnce(async () => {
      expect(api.CancelCloudflareSignIn).toHaveBeenCalledWith("s1");
      return session({ sessionId: "s2" });
    });
    await flow.start();
    expect(flow.sessionId).toBe("s2");
    await flow.cancel();
    expect(api.CancelCloudflareSignIn).toHaveBeenLastCalledWith("s2");
    expect(flow.sessionId).toBe("");
  });

  it("waits for an in-flight cancellation before restarting", async () => {
    const api = backend();
    const flow = new SignInSession(api);
    await flow.start();
    let finish!: () => void;
    api.CancelCloudflareSignIn.mockImplementationOnce(
      () =>
        new Promise<void>((resolve) => {
          finish = resolve;
        }),
    );
    const cancellation = flow.cancel();
    const restart = flow.start();
    await Promise.resolve();
    expect(api.StartCloudflareSignIn).toHaveBeenCalledTimes(1);
    finish();
    await cancellation;
    await restart;
    expect(api.StartCloudflareSignIn).toHaveBeenCalledTimes(2);
  });

  it("shares a pending start across repeated clicks", async () => {
    const api = backend();
    const flow = new SignInSession(api);
    const first = flow.start();
    const second = flow.start();
    expect(first).toBe(second);
    await first;
    expect(api.StartCloudflareSignIn).toHaveBeenCalledTimes(1);
  });

  it("cancels a start response that arrives after closing", async () => {
    const api = backend();
    let resolve!: (value: CloudflareSignIn) => void;
    api.StartCloudflareSignIn.mockImplementationOnce(
      () =>
        new Promise<CloudflareSignIn>((done) => {
          resolve = done;
        }),
    );
    const flow = new SignInSession(api);
    const pending = flow.start();
    await Promise.resolve();
    await flow.cancel();
    resolve(session());
    expect(await pending).toBeNull();
    expect(api.CancelCloudflareSignIn).toHaveBeenCalledWith("s1");
    expect(flow.sessionId).toBe("");
  });
});

describe("includesCloudflareAccount", () => {
  it("requires the original account for an existing connection", () => {
    const accounts = session({ accounts: [{ id: "a2", name: "Other" }] }).accounts;
    expect(includesCloudflareAccount(accounts, "a1")).toBe(false);
    expect(includesCloudflareAccount(accounts, "a2")).toBe(true);
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

  it("handles authorization that completed before start returned", () => {
    const ready = signInReducer(initialSignInState, {
      type: "started",
      session: session({ state: "authorized", accounts: [{ id: "a1", name: "First" }] }),
    });
    expect(ready.phase).toBe("ready");
    expect(ready.accountId).toBe("a1");
  });

  it("leaves waiting after cancellation", () => {
    const cancelled = signInReducer(started, {
      type: "updated",
      session: session({
        state: "failed",
        error: { code: "cancelled", message: "Operation cancelled" },
      }),
    });
    expect(cancelled.phase).toBe("error");
    expect(cancelled.error).toBe("Operation cancelled");
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
