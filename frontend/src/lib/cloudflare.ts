import type { CloudflareSignIn } from "./api";

/** Must match the redirect URI registered on the Cloudflare OAuth client. */
export const CLOUDFLARE_REDIRECT_URI = "http://127.0.0.1:8976/oauth/callback";

/** The scopes Bucky asks for; the last one is the sensitive permission that
 *  lets it mint the bucket-scoped R2 token. */
export const CLOUDFLARE_SCOPES = [
  "account-settings.read",
  "workers-r2.read",
  "workers-r2.write",
  "account-api-tokens.write",
];

export const CLOUDFLARE_CLIENTS_DOCS =
  "https://developers.cloudflare.com/fundamentals/oauth/create-an-oauth-client/";

export const CLOUDFLARE_DASHBOARD = "https://dash.cloudflare.com/";

/** Cloudflare only accepts a loopback HTTP callback for a public client. */
export function isLoopbackRedirect(value: string): boolean {
  try {
    const url = new URL(value.trim());
    if (url.protocol !== "http:" || url.username || url.password || url.search || url.hash) {
      return false;
    }
    if (!url.port) return false;
    return url.hostname === "localhost" || url.hostname === "127.0.0.1" || url.hostname === "[::1]";
  } catch {
    return false;
  }
}

export function normalizeScopes(values: string[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const value of values) {
    const scope = value.trim().toLowerCase();
    if (!scope || seen.has(scope) || !/^[a-z0-9._-]{1,64}$/.test(scope)) continue;
    seen.add(scope);
    out.push(scope);
  }
  return out;
}

export type SignInPhase = "idle" | "waiting" | "ready" | "connecting" | "error";

export interface SignInState {
  phase: SignInPhase;
  sessionId: string;
  authUrl: string;
  accounts: CloudflareSignIn["accounts"];
  accountId: string;
  error: string | null;
}

export const initialSignInState: SignInState = {
  phase: "idle",
  sessionId: "",
  authUrl: "",
  accounts: [],
  accountId: "",
  error: null,
};

export type SignInEvent =
  | { type: "started"; session: CloudflareSignIn }
  | { type: "updated"; session: CloudflareSignIn }
  | { type: "account"; accountId: string }
  | { type: "connecting" }
  | { type: "failed"; message: string }
  | { type: "reset" };

/** Keeps the dialog in step with the sign-in events the service emits. */
export function signInReducer(state: SignInState, event: SignInEvent): SignInState {
  switch (event.type) {
    case "started":
      return {
        phase: "waiting",
        sessionId: event.session.sessionId,
        authUrl: event.session.authUrl,
        accounts: [],
        accountId: "",
        error: null,
      };
    case "updated": {
      if (event.session.sessionId !== state.sessionId) return state;
      switch (event.session.state) {
        case "waiting":
          return state;
        case "authorized":
          return {
            ...state,
            phase: "ready",
            accounts: event.session.accounts,
            accountId: state.accountId || (event.session.accounts[0]?.id ?? ""),
            error: null,
          };
        default:
          return {
            ...state,
            phase: "error",
            error: event.session.error?.message ?? "",
          };
      }
    }
    case "account":
      return { ...state, accountId: event.accountId };
    case "connecting":
      return { ...state, phase: "connecting", error: null };
    case "failed":
      return { ...state, phase: "error", error: event.message };
    case "reset":
      return initialSignInState;
    default:
      return state;
  }
}
