import { AlertCircle, Check, LoaderCircle, X } from "lucide-react";
import {
  type ButtonHTMLAttributes,
  type ReactNode,
  useEffect,
  useId,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { errorMessage } from "../lib/api";

export function Button({
  variant = "secondary",
  className = "",
  children,
  type = "button",
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: "primary" | "secondary" | "ghost" | "danger";
  children: ReactNode;
}) {
  return (
    <button type={type} className={`button button-${variant} ${className}`} {...props}>
      {children}
    </button>
  );
}

export function IconButton({
  label,
  children,
  className = "",
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement> & { label: string; children: ReactNode }) {
  return (
    <button
      type="button"
      className={`icon-button ${className}`}
      aria-label={label}
      title={label}
      {...props}
    >
      {children}
    </button>
  );
}

export function Modal({
  title,
  subtitle,
  onClose,
  children,
  footer,
  wide = false,
  busy = false,
}: {
  title: string;
  subtitle?: string;
  onClose: () => void;
  children: ReactNode;
  footer?: ReactNode;
  wide?: boolean;
  busy?: boolean;
}) {
  const { t } = useTranslation();
  const titleID = useId();
  const ref = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    const dialog = ref.current;
    const previous = document.activeElement;
    dialog?.showModal();
    return () => {
      dialog?.close();
      if (previous instanceof HTMLElement && previous.isConnected) previous.focus();
    };
  }, []);
  return (
    <dialog
      ref={ref}
      className={`modal ${wide ? "modal-wide" : ""}`}
      aria-labelledby={titleID}
      onCancel={(event) => {
        event.preventDefault();
        if (!busy) onClose();
      }}
    >
      <header className="modal-header">
        <div>
          <h2 id={titleID}>{title}</h2>
          {subtitle && <p>{subtitle}</p>}
        </div>
        <IconButton label={t("close")} onClick={onClose} disabled={busy}>
          <X size={19} />
        </IconButton>
      </header>
      <div className="modal-body">{children}</div>
      {footer && <footer className="modal-footer">{footer}</footer>}
    </dialog>
  );
}

export function ErrorBanner({ message, onRetry }: { message: string; onRetry?: () => void }) {
  const { t } = useTranslation();
  return (
    <div className="error-banner" role="alert">
      <AlertCircle size={17} />
      <span>{message}</span>
      {onRetry && (
        <Button variant="ghost" onClick={onRetry}>
          {t("retry")}
        </Button>
      )}
    </div>
  );
}

export function Spinner({ label }: { label?: string }) {
  const { t } = useTranslation();
  return (
    <span className="loading-state" role="status">
      <LoaderCircle className="spin" size={18} />
      {label ?? t("loading")}
    </span>
  );
}

export function Brand({ small = false }: { small?: boolean }) {
  return (
    <span className={`brand ${small ? "brand-small" : ""}`}>
      <span className="brand-symbol" aria-hidden="true">
        <svg viewBox="0 0 32 32" fill="none">
          <title>Bucky</title>
          <path d="M8 6v20h11a7 7 0 0 0 0-14h-6V6H8Z" fill="currentColor" />
          <path d="m16 16 7-7" stroke="currentColor" strokeWidth="4" strokeLinecap="round" />
          <circle cx="24" cy="7" r="3" fill="currentColor" />
        </svg>
      </span>
      <span>
        Bucky<span className="brand-period">.</span>
      </span>
    </span>
  );
}

export function ConfirmDialog({
  title,
  message,
  confirmLabel,
  onConfirm,
  onClose,
  danger = false,
}: {
  title: string;
  message: string;
  confirmLabel: string;
  onConfirm: () => Promise<void>;
  onClose: () => void;
  danger?: boolean;
}) {
  const { t } = useTranslation();
  const [state, setState] = useAsyncState();
  return (
    <Modal
      title={title}
      onClose={onClose}
      busy={state.busy}
      footer={
        <>
          <Button onClick={onClose} disabled={state.busy}>
            {t("cancel")}
          </Button>
          <Button
            variant={danger ? "danger" : "primary"}
            disabled={state.busy}
            onClick={() => {
              void setState(onConfirm);
            }}
          >
            {state.busy ? <LoaderCircle size={16} className="spin" /> : <Check size={16} />}
            {confirmLabel}
          </Button>
        </>
      }
    >
      <p className="confirm-message">{message}</p>
      {state.error && <ErrorBanner message={state.error} />}
    </Modal>
  );
}

export function useAsyncState(): [
  { busy: boolean; error: string },
  (action: () => Promise<void>) => Promise<void>,
] {
  const [state, setState] = useState({ busy: false, error: "" });
  const lock = useRef(false);
  const run = async (action: () => Promise<void>) => {
    if (lock.current) return;
    lock.current = true;
    setState({ busy: true, error: "" });
    try {
      await action();
      setState({ busy: false, error: "" });
    } catch (error) {
      setState({ busy: false, error: errorMessage(error) });
    } finally {
      lock.current = false;
    }
  };
  return [state, run];
}
