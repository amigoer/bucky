import { Events, System, Window } from "@wailsio/runtime";
import { CopyIcon, MinusIcon, SquareIcon, XIcon } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";

// macOS keeps its native traffic lights; other platforms are frameless and draw
// their own window buttons.
export function WindowControls({ className }: { className?: string }) {
  const { t } = useTranslation();
  const [maximised, setMaximised] = useState(false);

  const read = useCallback(() => {
    Window.IsMaximised().then(setMaximised, () => setMaximised(false));
  }, []);

  useEffect(() => {
    read();
    // Linux emits no maximise event, so the state is re-read after toggling.
    if (!System.IsWindows()) return;
    const off = [
      Events.On("windows:WindowMaximise", () => setMaximised(true)),
      Events.On("windows:WindowUnMaximise", () => setMaximised(false)),
    ];
    return () => {
      for (const unsubscribe of off) unsubscribe();
    };
  }, [read]);

  const button =
    "grid h-7 w-8 place-items-center rounded-md text-muted-foreground transition-colors hover:bg-hover hover:text-foreground";
  return (
    <div className={cn("ml-2 flex flex-none items-center gap-0.5 border-l pl-2", className)}>
      <button
        type="button"
        className={button}
        aria-label={t("window.minimize")}
        onClick={() => void Window.Minimise().catch(() => {})}
      >
        <MinusIcon className="size-3.5" />
      </button>
      <button
        type="button"
        className={button}
        aria-label={maximised ? t("window.restore") : t("window.maximize")}
        onClick={() => void Window.ToggleMaximise().then(read, read)}
      >
        {maximised ? <CopyIcon className="size-3" /> : <SquareIcon className="size-3" />}
      </button>
      <button
        type="button"
        className={cn(button, "hover:bg-destructive-soft hover:text-destructive")}
        aria-label={t("window.close")}
        onClick={() => void Window.Close().catch(() => {})}
      >
        <XIcon className="size-3.5" />
      </button>
    </div>
  );
}
