import { CheckIcon, CopyIcon } from "lucide-react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Tip } from "@/components/ui/tooltip";
import { copyText } from "@/lib/actions";
import { useMediaQuery } from "@/lib/hooks";
import { type Accent, type Theme, usePrefs } from "@/lib/store";
import { ACCENTS, accentColor } from "@/lib/theme";
import { cn } from "@/lib/utils";

export function useResolvedDark(): boolean {
  const theme = usePrefs((s) => s.theme);
  const systemDark = useMediaQuery("(prefers-color-scheme: dark)");
  return theme === "dark" || (theme === "system" && systemDark);
}

export function Group({
  title,
  children,
  className,
}: {
  title?: string;
  children: ReactNode;
  className?: string;
}) {
  return (
    <section className="flex flex-col gap-2">
      {title ? (
        <h2 className="px-1 text-[12px] font-semibold text-muted-foreground">{title}</h2>
      ) : null}
      <div className={cn("flex flex-col divide-y rounded-xl bg-card shadow-card", className)}>
        {children}
      </div>
    </section>
  );
}

/** One setting: title and description on the left, its control on the right. */
export function Row({
  icon,
  title,
  description,
  children,
}: {
  icon?: ReactNode;
  title: ReactNode;
  description?: ReactNode;
  children?: ReactNode;
}) {
  return (
    <div className="flex min-h-14 items-center gap-6 px-4 py-3">
      <div className="flex min-w-0 flex-1 items-center gap-3">
        {icon ? (
          <span className="grid size-8 flex-none place-items-center rounded-lg bg-muted text-muted-foreground [&_svg]:size-4">
            {icon}
          </span>
        ) : null}
        <div className="flex min-w-0 flex-col gap-0.5">
          <span className="text-[13px] font-medium">{title}</span>
          {description ? (
            <span className="text-[12px] leading-relaxed text-muted-foreground">{description}</span>
          ) : null}
        </div>
      </div>
      {children ? <div className="flex flex-none items-center gap-2">{children}</div> : null}
    </div>
  );
}

/** A setting whose control needs the full width below its title. */
export function Block({
  title,
  description,
  children,
}: {
  title: ReactNode;
  description?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="flex flex-col gap-4 px-4 pt-3 pb-4">
      <div className="flex flex-col gap-0.5">
        <span className="text-[13px] font-medium">{title}</span>
        {description ? (
          <span className="text-[12px] leading-relaxed text-muted-foreground">{description}</span>
        ) : null}
      </div>
      {children}
    </div>
  );
}

export function CodeBlock({ code }: { code: string }) {
  const { t } = useTranslation();
  return (
    <div className="relative rounded-lg bg-[#17171a] shadow-[inset_0_0_0_1px_rgb(255_255_255/0.06)]">
      <pre
        data-selectable
        className="overflow-x-auto py-3 pr-12 pl-3.5 font-mono text-[12px] leading-relaxed whitespace-pre text-[#dcdce2]"
      >
        {code}
      </pre>
      <Tip label={t("common.copy")}>
        <Button
          variant="ghost"
          size="icon"
          className="absolute top-1.5 right-1.5 text-white/55 hover:bg-white/10 hover:text-white"
          aria-label={t("common.copy")}
          onClick={() => void copyText(code)}
        >
          <CopyIcon className="size-3.5" />
        </Button>
      </Tip>
    </div>
  );
}

const SCHEMES = {
  light: {
    window: "#ffffff",
    sidebar: "#f2f2f5",
    line: "rgb(0 0 0 / 0.07)",
    bar: "#dddde2",
    strong: "#c6c6cc",
  },
  dark: {
    window: "#1e1e20",
    sidebar: "#262629",
    line: "rgb(255 255 255 / 0.08)",
    bar: "#3b3b40",
    strong: "#56565c",
  },
};

const TRAFFIC = ["#ff5f57", "#febc2e", "#28c840"];

/** A miniature Bucky window drawn in the given scheme and accent. */
function MiniWindow({ dark, accent }: { dark: boolean; accent: Accent }) {
  const c = dark ? SCHEMES.dark : SCHEMES.light;
  const primary = accentColor(accent, dark);
  const { hue, chroma } = ACCENTS[accent];
  const soft = `oklch(${dark ? 0.63 : 0.56} ${chroma} ${hue} / ${dark ? 0.28 : 0.16})`;
  const folder = `oklch(${dark ? 0.55 : 0.8} ${chroma * 0.55} ${hue})`;
  return (
    <div className="flex size-full" style={{ background: c.window }}>
      <div
        className="flex w-[30%] flex-col gap-[5px] px-[7%] pt-[6%]"
        style={{ background: c.sidebar, borderRight: `1px solid ${c.line}` }}
      >
        <div className="mb-[3px] flex gap-[3px]">
          {TRAFFIC.map((color) => (
            <span key={color} className="size-[5px] rounded-full" style={{ background: color }} />
          ))}
        </div>
        <span className="h-[6px] w-full rounded-[2px]" style={{ background: soft }} />
        <span className="h-[4px] w-[70%] rounded-full" style={{ background: c.bar }} />
        <span className="h-[4px] w-[80%] rounded-full" style={{ background: c.bar }} />
        <span className="h-[4px] w-[60%] rounded-full" style={{ background: c.bar }} />
      </div>
      <div className="flex flex-1 flex-col">
        <div
          className="flex h-[22%] items-center justify-between px-[6%]"
          style={{ borderBottom: `1px solid ${c.line}` }}
        >
          <span className="h-[5px] w-[34%] rounded-full" style={{ background: c.strong }} />
          <span className="h-[8px] w-[20%] rounded-[3px]" style={{ background: primary }} />
        </div>
        <div className="flex flex-1 flex-col gap-[3px] px-[4%] pt-[5%]">
          {[0.62, 0.48, 0.7, 0.4].map((width, index) => (
            <div
              key={width}
              className="flex h-[11px] items-center gap-[5px] rounded-[3px] px-[4%]"
              style={{ background: index === 1 ? primary : "transparent" }}
            >
              <span
                className="size-[6px] flex-none rounded-[1.5px]"
                style={{ background: index === 1 ? "rgb(255 255 255 / 0.85)" : folder }}
              />
              <span
                className="h-[4px] rounded-full"
                style={{
                  width: `${width * 100}%`,
                  background: index === 1 ? "rgb(255 255 255 / 0.7)" : c.bar,
                }}
              />
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

export function ThemePreview({ theme, accent }: { theme: Theme; accent: Accent }) {
  if (theme !== "system") return <MiniWindow dark={theme === "dark"} accent={accent} />;
  return (
    <div className="relative size-full">
      <MiniWindow dark={false} accent={accent} />
      <div
        className="absolute inset-0"
        style={{ clipPath: "polygon(58% 0, 100% 0, 100% 100%, 42% 100%)" }}
      >
        <MiniWindow dark accent={accent} />
      </div>
    </div>
  );
}

export function ThemeCard({
  theme,
  label,
  icon,
  selected,
  accent,
  onSelect,
}: {
  theme: Theme;
  label: string;
  icon: ReactNode;
  selected: boolean;
  accent: Accent;
  onSelect: () => void;
}) {
  return (
    <button
      type="button"
      aria-pressed={selected}
      onClick={onSelect}
      className="group flex min-w-0 flex-col items-center gap-2.5 rounded-lg focus-visible:outline-offset-4"
    >
      <span
        className={cn(
          "block aspect-[16/10] w-full overflow-hidden rounded-lg transition-shadow duration-150",
          selected
            ? "shadow-[0_0_0_2px_var(--card),0_0_0_4px_var(--primary)]"
            : "shadow-card group-hover:shadow-card-hover",
        )}
      >
        <ThemePreview theme={theme} accent={accent} />
      </span>
      <span
        className={cn(
          "flex items-center gap-1.5 text-[12.5px] [&_svg]:size-3.5",
          selected ? "font-semibold text-foreground" : "text-muted-foreground",
        )}
      >
        {icon}
        {label}
      </span>
    </button>
  );
}

export function AccentPicker({
  value,
  order,
  labels,
  onChange,
}: {
  value: Accent;
  order: Accent[];
  labels: Record<Accent, string>;
  onChange: (accent: Accent) => void;
}) {
  const dark = useResolvedDark();
  return (
    <div className="flex items-center gap-2.5">
      {order.map((accent) => {
        const selected = accent === value;
        const color = accentColor(accent, dark);
        return (
          <Tip key={accent} label={labels[accent]}>
            <button
              type="button"
              aria-pressed={selected}
              aria-label={labels[accent]}
              onClick={() => onChange(accent)}
              className="grid size-5.5 place-items-center rounded-full transition-transform duration-150 ease-spring hover:scale-110 active:scale-95"
              style={{
                background: color,
                boxShadow: selected
                  ? `0 0 0 2px var(--card), 0 0 0 3.5px ${color}`
                  : "inset 0 0 0 1px rgb(0 0 0 / 0.12)",
              }}
            >
              {selected ? <CheckIcon className="size-3 text-white" strokeWidth={3} /> : null}
            </button>
          </Tip>
        );
      })}
    </div>
  );
}
