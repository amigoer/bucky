import {
  CheckIcon,
  CopyIcon,
  ExternalLinkIcon,
  LockIcon,
  MonitorIcon,
  MoonIcon,
  PieChartIcon,
  SunIcon,
  TerminalIcon,
} from "lucide-react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { PageHeader, PageTitle } from "@/components/page-header";
import { isMac, mod } from "@/components/platform";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Kbd } from "@/components/ui/kbd";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { copyText } from "@/lib/actions";
import { api } from "@/lib/api";
import { errorMessage } from "@/lib/errors";
import { useNav } from "@/lib/nav";
import { type Language, type Theme, usePrefs } from "@/lib/store";
import { cn } from "@/lib/utils";
import pkg from "../../../package.json";

const RATES = [0, 1, 2, 5, 10, 20, 50];

const MCP_SNIPPET = `{
  "mcpServers": {
    "bucky": { "command": "bucky", "args": ["mcp"] }
  }
}`;

const CLI_SNIPPET = `bucky ls r2://bucket/prefix --profile "Acme assets"`;

const SKILL_SNIPPET = `bucky skill print`;

const MCP_TOOLS = [
  "profiles",
  "buckets",
  "list",
  "stat",
  "find",
  "read_text",
  "presign",
  "usage",
  "transfers",
  "get_plan",
  "create_plan",
];

export function SettingsPage() {
  const { t } = useTranslation();
  const theme = usePrefs((s) => s.theme);
  const language = usePrefs((s) => s.language);
  const rate = usePrefs((s) => s.rate);
  const concurrency = usePrefs((s) => s.concurrency);
  const set = usePrefs((s) => s.set);
  const setPage = useNav((s) => s.setPage);

  const changeRate = async (value: string) => {
    const next = Number(value);
    set({ rate: next });
    try {
      await api.SetRate(next * 1024 * 1024);
    } catch (error) {
      toast.error(errorMessage(error));
    }
  };

  const changeConcurrency = async (value: string) => {
    const next = Number(value);
    set({ concurrency: next });
    try {
      await api.SetConcurrency(next);
    } catch (error) {
      toast.error(errorMessage(error));
    }
  };

  const shortcuts: { keys: string[]; label: string }[] = [
    { keys: [mod, "F"], label: t("keys.search") },
    { keys: [mod, "A"], label: t("keys.selectAll") },
    { keys: [mod, "1–3"], label: t("keys.views") },
    { keys: ["Space"], label: t("keys.quickLook") },
    { keys: ["↵"], label: t("keys.open") },
    { keys: isMac ? ["⌘", "⌫"] : ["Ctrl", "Backspace"], label: t("keys.delete") },
    { keys: isMac ? ["⌥", "⌘", "U"] : ["Ctrl", "Alt", "U"], label: t("keys.quickUpload") },
    { keys: isMac ? ["⇧", "⌘", "T"] : ["Ctrl", "Shift", "T"], label: t("keys.transfers") },
    { keys: [mod, ","], label: t("keys.settings") },
    { keys: isMac ? ["⌃", "⌘", "S"] : ["Ctrl", "Shift", "S"], label: t("keys.sidebar") },
    { keys: [mod, "[ / ]"], label: t("keys.history") },
    { keys: ["Esc"], label: t("keys.close") },
  ];

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <PageHeader>
        <PageTitle>{t("settings.title")}</PageTitle>
      </PageHeader>
      <div className="min-h-0 flex-1 overflow-y-auto">
        <div className="mx-auto flex w-full max-w-[820px] flex-col gap-6 px-6 py-5">
          <Section title={t("settings.appearance")}>
            <div className="grid grid-cols-3 gap-2.5">
              {(
                [
                  { value: "light", label: t("settings.light"), icon: SunIcon },
                  { value: "dark", label: t("settings.dark"), icon: MoonIcon },
                  { value: "system", label: t("settings.system"), icon: MonitorIcon },
                ] as const
              ).map(({ value, label, icon: Icon }) => (
                <button
                  key={value}
                  type="button"
                  onClick={() => set({ theme: value as Theme })}
                  className={cn(
                    "group flex flex-col gap-1.5 rounded-lg p-1.5 text-left transition-shadow",
                    theme === value ? "shadow-[0_0_0_2px_var(--primary)]" : "hover:bg-hover",
                  )}
                >
                  <ThemePreview theme={value} />
                  <span className="flex items-center gap-1.5 px-0.5 text-[11.5px] font-medium">
                    <Icon className="size-3 text-muted-foreground" />
                    {label}
                    {theme === value ? (
                      <CheckIcon className="ml-auto size-3 text-primary-text" strokeWidth={2.5} />
                    ) : null}
                  </span>
                </button>
              ))}
            </div>

            <div className="grid grid-cols-2 gap-2.5 pt-1">
              {(
                [
                  { value: "en", title: "English", sample: "Browse objects" },
                  { value: "zh", title: "简体中文", sample: "浏览对象" },
                ] as const
              ).map(({ value, title, sample }) => (
                <button
                  key={value}
                  type="button"
                  onClick={() => set({ language: value as Language })}
                  className={cn(
                    "flex items-center gap-2.5 rounded-lg border px-3 py-2 text-left transition-colors",
                    language === value ? "border-primary-line bg-primary-tint" : "hover:bg-hover",
                  )}
                >
                  <span className="grid size-7 flex-none place-items-center rounded-md bg-muted text-[12px] font-semibold">
                    Aa
                  </span>
                  <span className="flex min-w-0 flex-col">
                    <span className="text-[12px] font-medium">{title}</span>
                    <span className="truncate text-[10.5px] text-faint">{sample}</span>
                  </span>
                  {language === value ? (
                    <CheckIcon className="ml-auto size-3.5 text-primary-text" strokeWidth={2.5} />
                  ) : null}
                </button>
              ))}
            </div>
          </Section>

          <Section title={t("settings.transfers")}>
            <div className="grid grid-cols-2 gap-3">
              <div className="flex flex-col gap-1.5">
                <span className="text-[11.5px] font-medium text-muted-foreground">
                  {t("settings.rateLimit")}
                </span>
                <Select value={String(rate)} onValueChange={(value) => void changeRate(value)}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {RATES.map((value) => (
                      <SelectItem key={value} value={String(value)}>
                        {value === 0
                          ? t("transfers.unlimited")
                          : `${value} ${t("transfers.perSecond")}`}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <span className="text-[10.5px] leading-relaxed text-faint">
                  {t("settings.rateLimitHint")}
                </span>
              </div>
              <div className="flex flex-col gap-1.5">
                <span className="text-[11.5px] font-medium text-muted-foreground">
                  {t("settings.concurrency")}
                </span>
                <Select
                  value={String(concurrency)}
                  onValueChange={(value) => void changeConcurrency(value)}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {[1, 2, 3, 4, 5, 6, 7, 8].map((value) => (
                      <SelectItem key={value} value={String(value)}>
                        {value}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <span className="text-[10.5px] leading-relaxed text-faint">
                  {t("settings.concurrencyHint")}
                </span>
              </div>
            </div>
          </Section>

          <Section title={t("settings.usage")}>
            <div className="flex items-center gap-3">
              <PieChartIcon className="size-4 flex-none text-muted-foreground" />
              <span className="min-w-0 flex-1 text-[11.5px] text-muted-foreground">
                {t("settings.usageHint")}
              </span>
              <Button variant="secondary" size="sm" onClick={() => setPage("usage")}>
                <ExternalLinkIcon className="size-3.5" />
                {t("settings.openUsage")}
              </Button>
            </div>
          </Section>

          <Section title={t("settings.integrations")}>
            <Integration
              icon={<TerminalIcon />}
              title="MCP"
              description={t("settings.mcpDesc")}
              code={MCP_SNIPPET}
              tools={MCP_TOOLS}
              toolsLabel={t("settings.tools")}
            />
            <Integration
              icon={<TerminalIcon />}
              title="CLI"
              description={t("settings.cliDesc")}
              code={CLI_SNIPPET}
            />
            <Integration
              icon={<TerminalIcon />}
              title="Agent Skill"
              description={t("settings.skillDesc")}
              code={SKILL_SNIPPET}
            />
          </Section>

          <Section title={t("settings.shortcuts")}>
            <div className="grid grid-cols-2 gap-x-8 gap-y-1.5">
              {shortcuts.map((shortcut) => (
                <div
                  key={shortcut.label}
                  className="flex items-center justify-between gap-2 py-0.5 text-[11.5px]"
                >
                  <span className="truncate text-muted-foreground">{shortcut.label}</span>
                  <span className="flex flex-none gap-1">
                    {shortcut.keys.map((key) => (
                      <Kbd key={key}>{key}</Kbd>
                    ))}
                  </span>
                </div>
              ))}
            </div>
          </Section>

          <Section title={t("settings.privacy")}>
            <div className="flex items-start gap-2.5 text-[11.5px] leading-relaxed text-muted-foreground">
              <LockIcon className="mt-0.5 size-4 flex-none text-muted-foreground" />
              <span>{t("settings.privacyDesc")}</span>
            </div>
          </Section>

          <p className="pb-2 text-center text-[10.5px] text-faint">
            {t("settings.version", { version: pkg.version })}
          </p>
        </div>
      </div>
    </div>
  );
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-2">
      <h2 className="px-0.5 text-[11px] font-semibold text-faint">{title}</h2>
      <div className="flex flex-col gap-3 rounded-xl bg-card p-3.5 shadow-card">{children}</div>
    </section>
  );
}

function ThemePreview({ theme }: { theme: Theme }) {
  if (theme === "system") {
    return (
      <div className="flex h-[58px] w-full overflow-hidden rounded-md border shadow-[inset_0_1px_2px_rgb(0_0_0/0.05)]">
        <div className="flex w-1/2 bg-[#f4f4f6]">
          <div className="m-1.5 w-1/3 rounded-[2px] bg-white shadow-sm" />
        </div>
        <div className="flex w-1/2 bg-[#1e1e20]">
          <div className="m-1.5 w-1/3 rounded-[2px] bg-[#2b2b2e]" />
        </div>
      </div>
    );
  }
  const dark = theme === "dark";
  return (
    <div
      className="flex h-[58px] w-full overflow-hidden rounded-md border shadow-[inset_0_1px_2px_rgb(0_0_0/0.05)]"
      style={{ background: dark ? "#1e1e20" : "#ffffff" }}
    >
      <div className="w-[30%] border-r p-1.5" style={{ background: dark ? "#252528" : "#f4f4f6" }}>
        <div
          className="h-1 w-3/4 rounded-full"
          style={{ background: dark ? "#4a4a50" : "#d9d9de" }}
        />
        <div
          className="mt-1 h-1 w-1/2 rounded-full"
          style={{ background: dark ? "#3a3a3f" : "#e6e6ea" }}
        />
      </div>
      <div className="flex-1 p-1.5">
        <div
          className="h-1 w-1/2 rounded-full"
          style={{ background: dark ? "#4a4a50" : "#d9d9de" }}
        />
        <div className="mt-1.5 h-3 rounded-[2px]" style={{ background: "oklch(0.6 0.15 42)" }} />
        <div
          className="mt-1 h-1 w-2/3 rounded-full"
          style={{ background: dark ? "#3a3a3f" : "#e6e6ea" }}
        />
      </div>
    </div>
  );
}

function Integration({
  icon,
  title,
  description,
  code,
  tools,
  toolsLabel,
}: {
  icon: ReactNode;
  title: string;
  description: string;
  code: string;
  tools?: string[];
  toolsLabel?: string;
}) {
  const { t } = useTranslation();
  return (
    <div className="flex flex-col gap-2 border-b pb-3 last:border-b-0 last:pb-0">
      <div className="flex items-center gap-2">
        <span className="grid size-6 flex-none place-items-center rounded-md bg-muted text-muted-foreground [&_svg]:size-3.5">
          {icon}
        </span>
        <span className="text-[12px] font-semibold">{title}</span>
      </div>
      <p className="text-[11px] leading-relaxed text-muted-foreground">{description}</p>
      <div className="group relative rounded-lg bg-[#141416] px-3 py-2.5">
        <pre className="overflow-x-auto font-mono text-[10.5px] leading-relaxed whitespace-pre text-[#d7d7dc] select-text">
          {code}
        </pre>
        <Button
          variant="ghost"
          size="icon-sm"
          className="absolute top-1.5 right-1.5 text-white/50 hover:bg-white/10 hover:text-white"
          aria-label={t("common.copy")}
          onClick={() => void copyText(code)}
        >
          <CopyIcon className="size-3" />
        </Button>
      </div>
      {tools && tools.length > 0 ? (
        <div className="flex flex-wrap items-center gap-1">
          {toolsLabel ? <span className="mr-1 text-[10.5px] text-faint">{toolsLabel}</span> : null}
          {tools.map((tool) => (
            <Badge key={tool} variant="secondary" className="font-mono">
              {tool}
            </Badge>
          ))}
        </div>
      ) : null}
    </div>
  );
}
