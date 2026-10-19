import {
  ArrowRightLeftIcon,
  DownloadIcon,
  EyeIcon,
  FolderOpenIcon,
  KeyRoundIcon,
  Share2Icon,
  SquareArrowOutUpRightIcon,
  Trash2Icon,
} from "lucide-react";
import type * as React from "react";
import { useTranslation } from "react-i18next";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuShortcut,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import { copyText, downloadEntries } from "@/lib/actions";
import type { Entry } from "@/lib/entries";
import { type Location, useNav } from "@/lib/nav";

export function EntryContextMenu({
  entry,
  location,
  readOnly,
  children,
}: {
  entry: Entry;
  location: Location;
  readOnly: boolean;
  children: React.ReactElement;
}) {
  const { t } = useTranslation();
  const navigate = useNav((s) => s.navigate);
  const setQuickLook = useNav((s) => s.setQuickLook);
  const openShare = useNav((s) => s.openShare);
  const openPlan = useNav((s) => s.openPlan);

  const open = () => {
    if (entry.folder) navigate({ ...location, prefix: entry.key });
    else setQuickLook(entry.key);
  };

  return (
    <ContextMenu>
      <ContextMenuTrigger asChild>{children}</ContextMenuTrigger>
      <ContextMenuContent className="min-w-[196px]">
        <ContextMenuItem onSelect={open}>
          {entry.folder ? (
            <FolderOpenIcon className="text-muted-foreground" />
          ) : (
            <EyeIcon className="text-muted-foreground" />
          )}
          {entry.folder ? t("menu.open") : t("menu.quickLook")}
          <ContextMenuShortcut>{entry.folder ? "↵" : "Space"}</ContextMenuShortcut>
        </ContextMenuItem>
        <ContextMenuItem onSelect={() => void downloadEntries(location, [entry])}>
          <DownloadIcon className="text-muted-foreground" />
          {t("menu.download")}
        </ContextMenuItem>
        <ContextMenuItem onSelect={() => openShare({ location, entries: [entry] })}>
          <Share2Icon className="text-muted-foreground" />
          {t("menu.share")}
        </ContextMenuItem>
        <ContextMenuSeparator />
        <ContextMenuItem onSelect={() => void copyText(entry.key)}>
          <KeyRoundIcon className="text-muted-foreground" />
          {t("menu.copyKey")}
        </ContextMenuItem>
        {readOnly ? null : (
          <>
            <ContextMenuItem
              onSelect={() => openPlan({ operation: "copy", location, entries: [entry] })}
            >
              <SquareArrowOutUpRightIcon className="text-muted-foreground" />
              {t("menu.copyTo")}
            </ContextMenuItem>
            <ContextMenuItem
              onSelect={() => openPlan({ operation: "move", location, entries: [entry] })}
            >
              <ArrowRightLeftIcon className="text-muted-foreground" />
              {t("menu.moveTo")}
            </ContextMenuItem>
            <ContextMenuSeparator />
            <ContextMenuItem
              variant="destructive"
              onSelect={() => openPlan({ operation: "delete", location, entries: [entry] })}
            >
              <Trash2Icon />
              {t("menu.delete")}
              <ContextMenuShortcut>⌘⌫</ContextMenuShortcut>
            </ContextMenuItem>
          </>
        )}
      </ContextMenuContent>
    </ContextMenu>
  );
}
