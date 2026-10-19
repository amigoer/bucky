import type { Entry } from "@/lib/entries";
import type { Location } from "@/lib/nav";

export interface SelectModifiers {
  metaKey: boolean;
  ctrlKey: boolean;
  shiftKey: boolean;
}

export interface EntryViewProps {
  entries: Entry[];
  selection: Set<string>;
  focusKey?: string | null;
  onSelect: (entry: Entry, mods: SelectModifiers) => void;
  onOpen: (entry: Entry) => void;
  onLoadMore?: () => void;
  location: Location;
  readOnly: boolean;
}
