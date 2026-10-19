import { System } from "@wailsio/runtime";
import { isDesktop } from "@/lib/api";

function detectMac(): boolean {
  if (isDesktop()) {
    try {
      return System.IsMac();
    } catch {
      return false;
    }
  }
  return typeof navigator !== "undefined" && /mac/i.test(navigator.platform || navigator.userAgent);
}

export const isMac = detectMac();
export const mod = isMac ? "⌘" : "Ctrl+";
