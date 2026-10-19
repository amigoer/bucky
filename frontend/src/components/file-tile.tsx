import {
  ArchiveIcon,
  BracesIcon,
  CodeIcon,
  FileIcon,
  FileTextIcon,
  FolderIcon,
  ImageIcon,
  type LucideIcon,
  MusicIcon,
  TypeIcon,
  VideoIcon,
} from "lucide-react";
import { type Kind, tileStyle } from "@/lib/kinds";
import { cn } from "@/lib/utils";

export const KIND_ICON: Record<Kind, LucideIcon> = {
  folder: FolderIcon,
  image: ImageIcon,
  video: VideoIcon,
  audio: MusicIcon,
  pdf: FileTextIcon,
  text: FileTextIcon,
  code: CodeIcon,
  json: BracesIcon,
  archive: ArchiveIcon,
  font: TypeIcon,
  other: FileIcon,
};

/** Rounded tile with a kind icon; images get a striped placeholder. */
export function FileTile({
  kind,
  size = 28,
  selected = false,
  className,
}: {
  kind: Kind;
  size?: number;
  selected?: boolean;
  className?: string;
}) {
  const Icon = KIND_ICON[kind];
  const radius = size >= 28 ? 7 : 6;
  const style = selected
    ? { background: "rgb(255 255 255 / 0.22)", color: "var(--primary-foreground)" }
    : kind === "image"
      ? {
          background:
            "repeating-linear-gradient(135deg, var(--stripe) 0 4px, transparent 4px 8px), var(--tile-image)",
          color: "var(--kind-image-fg)",
        }
      : tileStyle(kind);
  return (
    <span
      className={cn("grid flex-none place-items-center", className)}
      style={{ width: size, height: size, borderRadius: radius, ...style }}
    >
      <Icon style={{ width: size * 0.54, height: size * 0.54 }} strokeWidth={1.8} />
    </span>
  );
}

const FOLDER_PATH =
  "M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z";

export function BigFolder({ size = 58 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" aria-hidden className="block">
      <path
        d={FOLDER_PATH}
        fill="var(--folder-fill)"
        stroke="var(--folder-stroke)"
        strokeWidth={1.1}
        strokeLinejoin="round"
      />
    </svg>
  );
}
