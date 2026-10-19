import i18n from "./i18n";

export interface ErrorInfo {
  code: string;
  message: string;
}

/** Wails rejects with the Go error message and puts the marshalled core.Error in `cause`. */
export function errorInfo(error: unknown): ErrorInfo {
  if (typeof error === "object" && error !== null) {
    const record = error as { cause?: unknown; code?: unknown; message?: unknown };
    let cause = record.cause;
    if (typeof cause === "string") {
      try {
        cause = JSON.parse(cause);
      } catch {
        cause = undefined;
      }
    }
    if (typeof cause === "object" && cause !== null) {
      const c = cause as { code?: unknown; message?: unknown };
      if (typeof c.code === "string") {
        return {
          code: c.code,
          message: typeof c.message === "string" ? c.message : String(record.message ?? ""),
        };
      }
    }
    if (typeof record.code === "string") {
      return { code: record.code, message: String(record.message ?? "") };
    }
    if (typeof record.message === "string") return { code: "", message: record.message };
  }
  return { code: "", message: String(error ?? "") };
}

export function errorCode(error: unknown): string {
  return errorInfo(error).code;
}

const KNOWN = new Set([
  "read_only",
  "conflict",
  "expired",
  "confirmation_required",
  "not_found",
  "forbidden",
  "configuration",
  "protocol",
  "local_io",
  "dialog",
  "clipboard",
  "cancelled",
  "timeout",
  "credentials_unavailable",
  "task_busy",
  "unsupported",
  "closed",
  "storage",
  "cleanup_failed",
  "network",
  "desktop_required",
]);

/**
 * User-facing text: validation messages are specific and already readable, so
 * they pass through; other codes get a localized explanation.
 */
export function errorMessage(error: unknown): string {
  const { code, message } = errorInfo(error);
  if (code === "invalid_argument" && message) return message;
  if (KNOWN.has(code)) return i18n.t(`errors.${code}` as "errors.conflict");
  if (code === "internal") return i18n.t("errors.internal");
  return message || i18n.t("errors.internal");
}

export function isCredentialError(error: unknown): boolean {
  const code = errorCode(error);
  return code === "credentials_unavailable" || code === "forbidden" || code === "configuration";
}
