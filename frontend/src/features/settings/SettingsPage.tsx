import {
  ArrowDownUpIcon,
  InfoIcon,
  KeyboardIcon,
  type LucideIcon,
  PaletteIcon,
  PlugIcon,
  Settings2Icon,
} from "lucide-react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { PageHeader, PageTitle } from "@/components/page-header";
import { type SettingsSection, useNav } from "@/lib/nav";
import { cn } from "@/lib/utils";
import {
  AboutSection,
  AppearanceSection,
  GeneralSection,
  IntegrationsSection,
  ShortcutsSection,
  TransfersSection,
} from "./sections";

const SECTIONS: { id: SettingsSection; icon: LucideIcon; content: () => ReactNode }[] = [
  { id: "general", icon: Settings2Icon, content: GeneralSection },
  { id: "appearance", icon: PaletteIcon, content: AppearanceSection },
  { id: "transfers", icon: ArrowDownUpIcon, content: TransfersSection },
  { id: "integrations", icon: PlugIcon, content: IntegrationsSection },
  { id: "shortcuts", icon: KeyboardIcon, content: ShortcutsSection },
  { id: "about", icon: InfoIcon, content: AboutSection },
];

export function SettingsPage() {
  const { t } = useTranslation();
  const current = useNav((s) => s.settingsSection);
  const openSettings = useNav((s) => s.openSettings);
  const section = SECTIONS.find((s) => s.id === current) ?? SECTIONS[0];
  const Content = section.content;

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <PageHeader>
        <PageTitle>{t("settings.title")}</PageTitle>
      </PageHeader>
      <div className="flex min-h-0 flex-1">
        <nav
          aria-label={t("settings.title")}
          className="flex w-[200px] flex-none flex-col gap-0.5 overflow-y-auto border-r p-3"
        >
          {SECTIONS.map(({ id, icon: Icon }) => {
            const active = id === section.id;
            return (
              <button
                key={id}
                type="button"
                aria-current={active ? "page" : undefined}
                onClick={() => openSettings(id)}
                className={cn(
                  "flex h-8 items-center gap-2.5 rounded-md px-2.5 text-left text-[13px] transition-colors",
                  active
                    ? "bg-primary-soft font-semibold text-primary-text"
                    : "text-muted-foreground hover:bg-hover hover:text-foreground",
                )}
              >
                <Icon className={cn("size-4 flex-none", active ? "text-primary" : "text-faint")} />
                <span className="truncate">{t(`settings.sections.${id}`)}</span>
              </button>
            );
          })}
        </nav>
        {/* Keyed so each section starts at the top. */}
        <div key={section.id} className="min-h-0 flex-1 animate-fade overflow-y-auto">
          <div className="mx-auto flex w-full max-w-[720px] flex-col gap-6 px-8 pt-7 pb-12">
            <header className="flex flex-col gap-1">
              <h1 className="text-[20px] font-[650] tracking-[-0.01em]">
                {t(`settings.sections.${section.id}`)}
              </h1>
              <p className="text-[13px] text-muted-foreground">
                {t(`settings.descriptions.${section.id}`)}
              </p>
            </header>
            <Content />
          </div>
        </div>
      </div>
    </div>
  );
}
