import {
  ArrowUpRightIcon,
  BotIcon,
  Columns3Icon,
  LayoutGridIcon,
  ListIcon,
  LockIcon,
  MonitorIcon,
  MoonIcon,
  PlugIcon,
  ScaleIcon,
  ShieldCheckIcon,
  SquareTerminalIcon,
  SunIcon,
} from "lucide-react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { Logo } from "@/components/brand";
import { isMac, mod } from "@/components/platform";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Kbd } from "@/components/ui/kbd";
import { Segmented } from "@/components/ui/segmented";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { api } from "@/lib/api";
import { errorMessage } from "@/lib/errors";
import { useNav } from "@/lib/nav";
import { type Accent, type Language, type Theme, usePrefs, type View } from "@/lib/store";
import pkg from "../../../package.json";
import { AccentPicker, Block, CodeBlock, Group, Row, ThemeCard } from "./parts";

const RATES = [0, 1, 2, 5, 10, 20, 50];
const CONCURRENCY = [1, 2, 3, 4, 5, 6, 7, 8];
const ACCENT_ORDER: Accent[] = ["blue", "indigo", "purple", "pink", "orange", "green", "graphite"];

const MCP_SNIPPET = `{
  "mcpServers": {
    "bucky": { "command": "bucky", "args": ["mcp"] }
  }
}`;

const CLI_SNIPPET = `bucky ls r2://bucket/prefix --profile "Acme assets"`;

const SKILL_SNIPPET = "bucky skill print";

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

export function GeneralSection() {
  const { t } = useTranslation();
  const language = usePrefs((s) => s.language);
  const view = usePrefs((s) => s.view);
  const set = usePrefs((s) => s.set);
  const setPage = useNav((s) => s.setPage);
  return (
    <>
      <Group>
        <Row title={t("settings.language")} description={t("settings.languageDesc")}>
          <Segmented<Language>
            aria-label={t("settings.language")}
            value={language}
            onValueChange={(value) => set({ language: value })}
            itemClassName="h-6.5 px-3.5 text-[12.5px]"
            options={[
              { value: "en", label: "English" },
              { value: "zh", label: "简体中文" },
            ]}
          />
        </Row>
        <Row title={t("settings.view")} description={t("settings.viewDesc", { keys: `${mod}1–3` })}>
          <Segmented<View>
            aria-label={t("settings.view")}
            value={view}
            onValueChange={(value) => set({ view: value })}
            itemClassName="h-6.5 px-3 text-[12.5px]"
            options={[
              { value: "list", label: t("settings.viewList"), icon: <ListIcon /> },
              { value: "grid", label: t("settings.viewIcons"), icon: <LayoutGridIcon /> },
              { value: "columns", label: t("settings.viewColumns"), icon: <Columns3Icon /> },
            ]}
          />
        </Row>
      </Group>
      <Group>
        <Row title={t("settings.usage")} description={t("settings.usageHint")}>
          <Button variant="secondary" onClick={() => setPage("usage")}>
            {t("settings.openUsage")}
            <ArrowUpRightIcon className="size-3.5" />
          </Button>
        </Row>
      </Group>
    </>
  );
}

export function AppearanceSection() {
  const { t } = useTranslation();
  const theme = usePrefs((s) => s.theme);
  const accent = usePrefs((s) => s.accent);
  const set = usePrefs((s) => s.set);
  const themes: { value: Theme; label: string; icon: ReactNode }[] = [
    { value: "light", label: t("settings.light"), icon: <SunIcon /> },
    { value: "dark", label: t("settings.dark"), icon: <MoonIcon /> },
    { value: "system", label: t("settings.system"), icon: <MonitorIcon /> },
  ];
  const labels = Object.fromEntries(
    ACCENT_ORDER.map((value) => [value, t(`settings.accents.${value}`)]),
  ) as Record<Accent, string>;
  return (
    <Group>
      <Block title={t("settings.theme")} description={t("settings.themeDesc")}>
        <div className="grid grid-cols-3 gap-5 px-1 pb-1">
          {themes.map(({ value, label, icon }) => (
            <ThemeCard
              key={value}
              theme={value}
              label={label}
              icon={icon}
              accent={accent}
              selected={theme === value}
              onSelect={() => set({ theme: value })}
            />
          ))}
        </div>
      </Block>
      <Row title={t("settings.accent")} description={t("settings.accentDesc")}>
        <AccentPicker
          value={accent}
          order={ACCENT_ORDER}
          labels={labels}
          onChange={(value) => set({ accent: value })}
        />
      </Row>
    </Group>
  );
}

export function TransfersSection() {
  const { t } = useTranslation();
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

  return (
    <>
      <Group>
        <Row title={t("settings.rateLimit")} description={t("settings.rateLimitHint")}>
          <Select value={String(rate)} onValueChange={(value) => void changeRate(value)}>
            <SelectTrigger className="w-36" aria-label={t("settings.rateLimit")}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {RATES.map((value) => (
                <SelectItem key={value} value={String(value)}>
                  {value === 0 ? t("transfers.unlimited") : `${value} ${t("transfers.perSecond")}`}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </Row>
        <Row title={t("settings.concurrency")} description={t("settings.concurrencyHint")}>
          <Select
            value={String(concurrency)}
            onValueChange={(value) => void changeConcurrency(value)}
          >
            <SelectTrigger className="w-36" aria-label={t("settings.concurrency")}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {CONCURRENCY.map((value) => (
                <SelectItem key={value} value={String(value)}>
                  {value}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </Row>
      </Group>
      <Group>
        <Row title={t("settings.transferQueue")} description={t("settings.transferNote")}>
          <Button variant="secondary" onClick={() => setPage("transfers")}>
            {t("settings.openTransfers")}
            <ArrowUpRightIcon className="size-3.5" />
          </Button>
        </Row>
      </Group>
    </>
  );
}

function Integration({
  icon,
  title,
  description,
  code,
  children,
}: {
  icon: ReactNode;
  title: string;
  description: string;
  code: string;
  children?: ReactNode;
}) {
  return (
    <section className="flex flex-col gap-3.5 rounded-xl bg-card p-4 shadow-card">
      <div className="flex items-start gap-3">
        <span className="grid size-8 flex-none place-items-center rounded-lg bg-muted text-muted-foreground [&_svg]:size-4">
          {icon}
        </span>
        <div className="flex min-w-0 flex-col gap-0.5">
          <span className="text-[13px] font-semibold">{title}</span>
          <p className="text-[12px] leading-relaxed text-muted-foreground">{description}</p>
        </div>
      </div>
      <CodeBlock code={code} />
      {children}
    </section>
  );
}

export function IntegrationsSection() {
  const { t } = useTranslation();
  return (
    <>
      <Integration
        icon={<PlugIcon />}
        title="MCP"
        description={t("settings.mcpDesc")}
        code={MCP_SNIPPET}
      >
        <div className="flex flex-wrap items-center gap-1.5">
          <span className="mr-1 text-[12px] text-muted-foreground">{t("settings.tools")}</span>
          {MCP_TOOLS.map((tool) => (
            <Badge key={tool} variant="secondary" className="font-mono">
              {tool}
            </Badge>
          ))}
        </div>
      </Integration>
      <Integration
        icon={<SquareTerminalIcon />}
        title="CLI"
        description={t("settings.cliDesc")}
        code={CLI_SNIPPET}
      />
      <Integration
        icon={<BotIcon />}
        title="Agent Skill"
        description={t("settings.skillDesc")}
        code={SKILL_SNIPPET}
      />
    </>
  );
}

export function ShortcutsSection() {
  const { t } = useTranslation();
  const groups: { title: string; items: { keys: string[]; label: string }[] }[] = [
    {
      title: t("settings.shortcutGroups.browse"),
      items: [
        { keys: [mod, "F"], label: t("keys.search") },
        { keys: [mod, "1–3"], label: t("keys.views") },
        { keys: ["↵"], label: t("keys.open") },
        { keys: [mod, "[ / ]"], label: t("keys.history") },
      ],
    },
    {
      title: t("settings.shortcutGroups.files"),
      items: [
        { keys: [mod, "A"], label: t("keys.selectAll") },
        { keys: ["Space"], label: t("keys.quickLook") },
        { keys: isMac ? ["⌘", "⌫"] : ["Ctrl", "Backspace"], label: t("keys.delete") },
        { keys: ["Esc"], label: t("keys.close") },
      ],
    },
    {
      title: t("settings.shortcutGroups.app"),
      items: [
        { keys: isMac ? ["⌥", "⌘", "U"] : ["Ctrl", "Alt", "U"], label: t("keys.quickUpload") },
        { keys: isMac ? ["⇧", "⌘", "T"] : ["Ctrl", "Shift", "T"], label: t("keys.transfers") },
        { keys: [mod, ","], label: t("keys.settings") },
        { keys: isMac ? ["⌃", "⌘", "S"] : ["Ctrl", "Shift", "S"], label: t("keys.sidebar") },
      ],
    },
  ];
  return (
    <>
      {groups.map((group) => (
        <Group key={group.title} title={group.title}>
          {group.items.map((item) => (
            <div
              key={item.label}
              className="flex h-11 items-center justify-between gap-4 px-4 text-[13px]"
            >
              <span className="truncate">{item.label}</span>
              <span className="flex flex-none gap-1">
                {item.keys.map((key) => (
                  <Kbd key={key}>{key}</Kbd>
                ))}
              </span>
            </div>
          ))}
        </Group>
      ))}
    </>
  );
}

export function AboutSection() {
  const { t } = useTranslation();
  return (
    <>
      <section className="flex flex-col items-center gap-2 rounded-xl bg-card px-6 pt-8 pb-7 text-center shadow-card">
        <Logo className="mb-1 size-16 rounded-[16px]" />
        <span className="text-[20px] font-bold tracking-[-0.02em]">Bucky</span>
        <span className="text-[13px] text-muted-foreground">{t("settings.tagline")}</span>
        <Badge variant="secondary" className="mt-1 font-mono">
          {t("settings.version", { version: pkg.version })}
        </Badge>
      </section>
      <Group>
        <Row
          icon={<LockIcon />}
          title={t("settings.privacy")}
          description={t("settings.privacyDesc")}
        />
        <Row
          icon={<ShieldCheckIcon />}
          title={t("settings.safety")}
          description={t("settings.safetyDesc")}
        />
        <Row icon={<ScaleIcon />} title={t("settings.license")} description="Apache License 2.0" />
      </Group>
    </>
  );
}
