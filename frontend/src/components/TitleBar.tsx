import { Events, System, Window } from "@wailsio/runtime";
import { Copy, Minus, Square, X } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Brand } from "./ui";

const mac = System.IsMac();

export function TitleBar() {
  return (
    <header className={`titlebar${mac ? " titlebar--mac" : ""}`}>
      <Brand small />
      {mac ? null : <WindowControls />}
    </header>
  );
}

// macOS keeps its native traffic lights; the other platforms are frameless and
// draw their own window buttons here.
function WindowControls() {
  const { t } = useTranslation();
  const [maximised, setMaximised] = useState(false);

  const readMaximised = useCallback(() => {
    Window.IsMaximised().then(setMaximised, () => setMaximised(false));
  }, []);

  useEffect(() => {
    readMaximised();
    // Linux emits no maximise event, so the state is re-read after toggling.
    if (!System.IsWindows()) return;
    const off = [
      Events.On("windows:WindowMaximise", () => setMaximised(true)),
      Events.On("windows:WindowUnMaximise", () => setMaximised(false)),
    ];
    return () => {
      for (const unsubscribe of off) unsubscribe();
    };
  }, [readMaximised]);

  const restore = maximised ? t("restore") : t("maximize");

  return (
    <div className="window-controls">
      <button
        type="button"
        className="window-control"
        title={t("minimize")}
        aria-label={t("minimize")}
        onClick={() => void Window.Minimise().catch(() => {})}
      >
        <Minus size={13} aria-hidden />
      </button>
      <button
        type="button"
        className="window-control"
        title={restore}
        aria-label={restore}
        onClick={() => void Window.ToggleMaximise().then(readMaximised, readMaximised)}
      >
        {maximised ? <Copy size={12} aria-hidden /> : <Square size={13} aria-hidden />}
      </button>
      <button
        type="button"
        className="window-control window-control-close"
        title={t("close")}
        aria-label={t("close")}
        onClick={() => void Window.Close().catch(() => {})}
      >
        <X size={13} aria-hidden />
      </button>
    </div>
  );
}
