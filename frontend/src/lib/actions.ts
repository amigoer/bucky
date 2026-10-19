import { toast } from "sonner";
import { api, type ObjectRef, ref } from "./api";
import type { Entry } from "./entries";
import { errorMessage } from "./errors";
import i18n from "./i18n";
import { type Location, useNav } from "./nav";

export const MAX_PLAN_OBJECTS = 1000;

export class TooManyObjectsError extends Error {
  constructor() {
    super("too many objects");
  }
}

/** Expands folders into every object below them, including folder markers. */
export async function expandEntries(location: Location, entries: Entry[]): Promise<ObjectRef[]> {
  const out: ObjectRef[] = [];
  const seen = new Set<string>();
  const add = (key: string) => {
    if (seen.has(key)) return;
    seen.add(key);
    out.push(ref(location.profileId, location.bucket, key));
    if (out.length > MAX_PLAN_OBJECTS) throw new TooManyObjectsError();
  };
  for (const entry of entries) {
    if (!entry.folder) {
      add(entry.key);
      continue;
    }
    let cursor = "";
    for (;;) {
      const result = await api.Find({
        profileId: location.profileId,
        bucket: location.bucket,
        prefix: entry.key,
        contains: "",
        before: "",
        minSize: 0,
        maxSize: 0,
        limit: MAX_PLAN_OBJECTS + 1,
        cursor,
      });
      for (const obj of result.objects ?? []) add(obj.key);
      if (result.complete || !result.nextCursor) break;
      cursor = result.nextCursor;
    }
  }
  return out;
}

export function failureMessage(error: unknown): string {
  if (error instanceof TooManyObjectsError) return i18n.t("plan.tooMany");
  return errorMessage(error);
}

export async function copyText(text: string, quiet = false) {
  try {
    await api.CopyText(text);
  } catch {
    try {
      await navigator.clipboard.writeText(text);
    } catch (error) {
      toast.error(errorMessage(error));
      return false;
    }
  }
  if (!quiet) toast.success(i18n.t("toast.copied"));
  return true;
}

export async function downloadEntries(location: Location, entries: Entry[]) {
  try {
    if (entries.length === 1 && !entries[0].folder) {
      const entry = entries[0];
      const localPath = await api.PickDownload(entry.name.split("/").pop() || entry.name);
      if (!localPath) return;
      await api.StartTransfer({
        direction: "download",
        object: ref(location.profileId, location.bucket, entry.key),
        localPath,
      });
      toast.success(i18n.t("toast.queued", { count: 1 }));
      return;
    }
    const dir = await api.PickDirectory();
    if (!dir) return;
    const refs = await expandEntries(location, entries);
    if (refs.length === 0) {
      toast(i18n.t("toast.nothingToDownload"));
      return;
    }
    const results = await api.DownloadTo(dir, refs, location.prefix);
    const failed = results.filter((r) => r.error);
    const queued = results.filter((r) => r.task).length;
    if (failed.length > 0) {
      toast.error(
        i18n.t("toast.downloadPartial", {
          failed: failed.length,
          total: results.length,
          message: errorMessage(failed[0].error),
        }),
      );
    } else if (queued > 0) {
      toast.success(i18n.t("toast.queued", { count: queued }));
    }
  } catch (error) {
    toast.error(failureMessage(error));
  }
}

export async function uploadPaths(location: Location, paths: string[]) {
  if (paths.length === 0) return;
  try {
    const results = await api.UploadPaths(
      location.profileId,
      location.bucket,
      location.prefix,
      paths,
    );
    const tasks = results.filter((r) => r.task).length;
    const plans = results.flatMap((r) => (r.plan ? [r.plan] : []));
    if (tasks > 0) toast.success(i18n.t("toast.queued", { count: tasks }));
    if (plans.length > 0) {
      toast(i18n.t("toast.reviewPlans", { count: plans.length }));
      useNav
        .getState()
        .queuePlans(plans.map((plan) => ({ operation: "upload", location, entries: [], plan })));
    }
  } catch (error) {
    toast.error(errorMessage(error));
  }
}

export async function pickAndUpload(location: Location, directory: boolean) {
  try {
    const paths = await api.PickFiles(directory);
    await uploadPaths(location, paths);
  } catch (error) {
    toast.error(errorMessage(error));
  }
}
