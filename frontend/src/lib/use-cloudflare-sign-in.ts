import { useCallback, useEffect, useReducer, useRef, useState } from "react";
import { api, type CloudflareSignIn, events } from "./api";
import { initialSignInState, SignInSession, signInReducer } from "./cloudflare";
import { errorMessage } from "./errors";

export function useCloudflareSignIn() {
  const [session] = useState(() => new SignInSession(api));
  const [state, dispatch] = useReducer(signInReducer, initialSignInState);
  const [starting, setStarting] = useState(false);
  const mounted = useRef(false);
  const latest = useRef<CloudflareSignIn | null>(null);

  useEffect(() => {
    mounted.current = true;
    const unsubscribe = events.onCloudflareSignIn((update) => {
      latest.current = update;
      if (session.sessionId === update.sessionId) {
        dispatch({ type: "updated", session: update });
      }
    });
    return () => {
      mounted.current = false;
      unsubscribe();
      void session.cancel().catch(() => {});
    };
  }, [session]);

  const start = useCallback(async () => {
    setStarting(true);
    dispatch({ type: "reset" });
    latest.current = null;
    try {
      const update = await session.start();
      if (!mounted.current || !update) return;
      dispatch({ type: "started", session: update });
      // Authorization can finish before the start call reaches the renderer.
      const received = latest.current as CloudflareSignIn | null;
      if (received?.sessionId === update.sessionId) {
        dispatch({ type: "updated", session: received });
      }
    } catch (cause) {
      if (mounted.current) dispatch({ type: "failed", message: errorMessage(cause) });
      throw cause;
    } finally {
      if (mounted.current) setStarting(false);
    }
  }, [session]);

  const cancel = useCallback(async () => {
    dispatch({ type: "reset" });
    await session.cancel();
  }, [session]);

  const fail = useCallback(
    (message: string) => {
      dispatch({ type: "failed", message });
      void session.cancel().catch(() => {});
    },
    [session],
  );

  const isCurrent = useCallback((id: string) => session.sessionId === id, [session]);

  const selectAccount = (accountId: string) => dispatch({ type: "account", accountId });

  return { state, starting, start, cancel, fail, isCurrent, selectAccount };
}
