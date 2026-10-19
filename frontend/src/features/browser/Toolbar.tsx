import {
  ArrowDownUpIcon,
  CheckIcon,
  ChevronDownIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  Columns3Icon,
  LayoutGridIcon,
  ListIcon,
  UploadIcon,
  ZapIcon,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { PageHeader } from "@/components/page-header";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Segmented } from "@/components/ui/segmented";
import { Tip } from "@/components/ui/tooltip";
import { pickAndUpload } from "@/lib/actions";
import type { SortKey } from "@/lib/entries";
import { type Location, useNav } from "@/lib/nav";
import { usePrefs, type View } from "@/lib/store";
import { Breadcrumbs } from "./Breadcrumbs";
import { SearchBox } from "./SearchBox";

export function Toolbar({ location, readOnly }: { location: Location; readOnly: boolean }) {
  const { t } = useTranslation();
  const canGoBack = useNav((s) => s.backStack.length > 0);
  const canGoForward = useNav((s) => s.forwardStack.length > 0);
  const goBack = useNav((s) => s.goBack);
  const goForward = useNav((s) => s.goForward);
  const navigate = useNav((s) => s.navigate);
  const openQuickUpload = useNav((s) => s.openQuickUpload);
  const view = usePrefs((s) => s.view);
  const set = usePrefs((s) => s.set);
  const sortKey = usePrefs((s) => s.sortKey);
  const sortDir = usePrefs((s) => s.sortDir);

  return (
    <PageHeader className="gap-1.5">
      <div className="flex flex-none items-center gap-0.5">
        <Tip label={t("browser.back")}>
          <Button
            variant="ghost"
            size="icon"
            aria-label={t("browser.back")}
            disabled={!canGoBack}
            onClick={goBack}
          >
            <ChevronLeftIcon className="size-4" strokeWidth={2.2} />
          </Button>
        </Tip>
        <Tip label={t("browser.forward")}>
          <Button
            variant="ghost"
            size="icon"
            aria-label={t("browser.forward")}
            disabled={!canGoForward}
            onClick={goForward}
          >
            <ChevronRightIcon className="size-4" strokeWidth={2.2} />
          </Button>
        </Tip>
      </div>

      <Breadcrumbs location={location} onNavigate={(prefix) => navigate({ ...location, prefix })} />

      <div className="ml-auto flex flex-none items-center gap-2">
        <SearchBox location={location} />
        <Segmented<View>
          value={view}
          onValueChange={(next) => set({ view: next })}
          className="w-[104px]"
          itemClassName="px-0"
          aria-label={t("browser.list")}
          options={[
            {
              value: "list",
              icon: <ListIcon />,
              title: t("browser.list"),
              label: <span className="sr-only">{t("browser.list")}</span>,
            },
            {
              value: "grid",
              icon: <LayoutGridIcon />,
              title: t("browser.grid"),
              label: <span className="sr-only">{t("browser.grid")}</span>,
            },
            {
              value: "columns",
              icon: <Columns3Icon />,
              title: t("browser.columns"),
              label: <span className="sr-only">{t("browser.columns")}</span>,
            },
          ]}
        />

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon" aria-label={t("browser.sort")}>
              <ArrowDownUpIcon className="size-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="min-w-[168px]">
            <DropdownMenuRadioGroup
              value={sortKey}
              onValueChange={(value) => set({ sortKey: value as SortKey })}
            >
              <DropdownMenuRadioItem value="name">{t("browser.sortName")}</DropdownMenuRadioItem>
              <DropdownMenuRadioItem value="size">{t("browser.sortSize")}</DropdownMenuRadioItem>
              <DropdownMenuRadioItem value="modified">
                {t("browser.sortModified")}
              </DropdownMenuRadioItem>
            </DropdownMenuRadioGroup>
            <DropdownMenuSeparator />
            <DropdownMenuItem onSelect={() => set({ sortDir: sortDir === 1 ? -1 : 1 })}>
              <CheckIcon className="opacity-70" />
              {sortDir === 1 ? t("browser.ascending") : t("browser.descending")}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>

        {readOnly ? (
          <Tip label={t("browser.readOnlyTip")}>
            <span>
              <Button variant="secondary" size="sm" disabled>
                <UploadIcon className="size-3.5" />
                {t("browser.upload")}
              </Button>
            </span>
          </Tip>
        ) : (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="secondary" size="sm">
                <UploadIcon className="size-3.5" />
                {t("browser.upload")}
                <ChevronDownIcon className="size-3 opacity-60" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="min-w-[200px]">
              <DropdownMenuItem onSelect={() => void pickAndUpload(location, false)}>
                <UploadIcon className="text-muted-foreground" />
                {t("browser.uploadFiles")}
              </DropdownMenuItem>
              <DropdownMenuItem onSelect={() => void pickAndUpload(location, true)}>
                <UploadIcon className="text-muted-foreground" />
                {t("browser.uploadFolder")}
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem onSelect={() => openQuickUpload({})}>
                <ZapIcon className="text-muted-foreground" />
                {t("quick.title")}
                <DropdownMenuShortcut>⌥⌘U</DropdownMenuShortcut>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>
    </PageHeader>
  );
}
