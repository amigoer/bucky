import { CheckIcon } from "lucide-react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { CloudflareMark } from "@/components/cloudflare-mark";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";

export type Tone = "accent" | "success" | "warn" | "muted";

const TILE_TONE: Record<Tone, string> = {
  accent: "bg-primary-soft text-primary-text",
  success: "bg-success-soft text-success-text",
  warn: "bg-warning-soft text-warning-text",
  muted: "bg-muted text-muted-foreground",
};

const NOTE_TONE: Record<Tone, string> = {
  accent: "bg-primary-soft text-primary-text",
  success: "bg-success-soft text-success-text",
  warn: "bg-warning-soft text-warning-text",
  muted: "bg-field text-muted-foreground",
};

/** Provider mark and name at the top of every step. */
export function ProviderHeading() {
  const { t } = useTranslation();
  return (
    <div className="flex items-center gap-3">
      <CloudflareMark size={34} />
      <span className="flex flex-col">
        <span className="text-[15px] font-semibold">{t("connection.r2")}</span>
        <span className="text-[11.5px] text-faint">{t("connection.provider")}</span>
      </span>
    </div>
  );
}

/** Rounded icon tile; the app repeats this pattern inline rather than sharing one. */
export function IconTile({
  icon,
  tone = "muted",
  className,
}: {
  icon: ReactNode;
  tone?: Tone;
  className?: string;
}) {
  return (
    <span
      className={cn(
        "grid size-10 flex-none place-items-center rounded-[11px] [&_svg]:size-4.5",
        TILE_TONE[tone],
        className,
      )}
    >
      {icon}
    </span>
  );
}

/** Label, control and optional hint — the one field layout every connection form uses. */
export function Field({
  label,
  hint,
  children,
  className,
}: {
  label: ReactNode;
  hint?: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("flex min-w-0 flex-col gap-1.5", className)}>
      <span className="text-[11.5px] font-medium text-muted-foreground">{label}</span>
      {children}
      {hint ? <span className="text-[10.5px] leading-relaxed text-faint">{hint}</span> : null}
    </div>
  );
}

/** Inline icon and text, for prerequisites, security promises and short asides. */
export function Note({
  icon,
  children,
  tone = "muted",
  className,
}: {
  icon?: ReactNode;
  children: ReactNode;
  tone?: Tone;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "flex items-start gap-2.5 rounded-xl px-3.5 py-2.5 text-[12px] leading-relaxed",
        NOTE_TONE[tone],
        className,
      )}
    >
      {icon ? <span className="mt-px flex-none [&_svg]:size-4">{icon}</span> : null}
      <div className="min-w-0">{children}</div>
    </div>
  );
}

/** Icon tile plus title and description, used for the capability lists. */
export function CapabilityRow({
  icon,
  tone,
  title,
  description,
}: {
  icon: ReactNode;
  tone: Tone;
  title: ReactNode;
  description: ReactNode;
}) {
  return (
    <div className="flex items-start gap-3">
      <IconTile icon={icon} tone={tone} />
      <div className="flex min-w-0 flex-col gap-1">
        <span className="text-[13.5px] font-semibold">{title}</span>
        <span className="text-[12px] leading-relaxed text-muted-foreground">{description}</span>
      </div>
    </div>
  );
}

/** One step of the progress list shown while Cloudflare authorization runs. */
export function StepRow({
  index,
  title,
  description,
  state,
}: {
  index: number;
  title: ReactNode;
  description?: ReactNode;
  state: "done" | "active" | "todo";
}) {
  return (
    <div className="flex items-start gap-3">
      <span
        className={cn(
          "grid size-5.5 flex-none place-items-center rounded-full text-[11px] font-semibold",
          state === "done" && "bg-success-soft text-success-text",
          state === "active" && "bg-primary text-primary-foreground",
          state === "todo" && "bg-muted text-faint",
        )}
      >
        {state === "done" ? <CheckIcon className="size-3" strokeWidth={3} /> : index}
      </span>
      <div className="flex min-w-0 flex-col gap-0.5">
        <span
          className={cn(
            "text-[13.5px] font-medium",
            state === "todo" ? "text-faint" : "text-foreground",
          )}
        >
          {title}
        </span>
        {description ? (
          <span className="text-[12px] leading-relaxed text-muted-foreground">{description}</span>
        ) : null}
      </div>
    </div>
  );
}

/** Bordered row with a label, an explanation and a switch on the right. */
export function SwitchRow({
  id,
  title,
  description,
  checked,
  onCheckedChange,
}: {
  id: string;
  title: ReactNode;
  description: ReactNode;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
}) {
  return (
    <div className="flex items-center justify-between gap-3 rounded-xl border px-3.5 py-2.5">
      <div className="flex min-w-0 flex-col">
        <Label htmlFor={id} className="text-[13px]">
          {title}
        </Label>
        <span className="text-[11.5px] leading-relaxed text-muted-foreground">{description}</span>
      </div>
      <Switch id={id} checked={checked} onCheckedChange={onCheckedChange} className="flex-none" />
    </div>
  );
}

/** White card on the grey right column. */
export function Panel({
  title,
  action,
  children,
  className,
}: {
  title?: ReactNode;
  action?: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  return (
    <section
      className={cn("flex flex-col gap-3.5 rounded-2xl bg-card p-4.5 shadow-card", className)}
    >
      {title ? (
        <header className="flex items-center gap-2">
          <h2 className="text-[12.5px] font-semibold text-muted-foreground">{title}</h2>
          <span className="flex-1" />
          {action}
        </header>
      ) : null}
      {children}
    </section>
  );
}

/** Left column: centred scrollable step content, with an optional pinned footer. */
export function StepBody({ children, footer }: { children: ReactNode; footer?: ReactNode }) {
  return (
    <>
      <div className="flex min-h-0 flex-1 flex-col overflow-y-auto px-10 py-10">
        <div className="mx-auto my-auto flex w-full max-w-[480px] flex-col gap-7">{children}</div>
      </div>
      {footer ? (
        <div className="flex flex-none items-center gap-2 border-t bg-sidebar px-10 py-3">
          {footer}
        </div>
      ) : null}
    </>
  );
}

export function LeftColumn({ children }: { children: ReactNode }) {
  return <div className="flex min-w-0 flex-1 flex-col">{children}</div>;
}

/** Right column: the reference and result surface, hidden on narrow windows. */
export function RightColumn({ children }: { children: ReactNode }) {
  return (
    <aside className="flex w-[46%] max-w-[620px] min-w-[340px] flex-none flex-col overflow-y-auto border-l bg-sidebar px-7 py-8 max-[900px]:hidden">
      <div className="my-auto flex flex-col gap-4">{children}</div>
    </aside>
  );
}
