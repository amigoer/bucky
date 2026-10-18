// Static imports avoid a cyclic chunk graph that WebKit evaluates out of order.
import { Events } from "@wailsio/runtime";
import { DesktopService } from "../../bindings/github.com/amigoer/bucky/internal/app/index";
import * as core from "../../bindings/github.com/amigoer/bucky/internal/core/models";
import * as service from "../../bindings/github.com/amigoer/bucky/internal/service/models";

export type Profile = core.Profile;
export type Bucket = core.Bucket;
export type ObjectInfo = core.ObjectInfo;
export type ObjectRef = core.ObjectRef;
export type ListRequest = core.ListRequest;
export type ListPage = core.ListPage;
export type OperationPlan = core.OperationPlan;
export type PlanRequest = core.PlanRequest;
export type TransferTask = core.TransferTask;
export type TransferRequest = core.TransferRequest;
export type ShareLink = core.ShareLink;
export type TextPreview = core.TextPreview;
export type Usage = core.Usage;
export type FindRequest = core.FindRequest;
export type ProfileInput = service.ProfileInput;
export type TransferResult = service.TransferResult;

interface NativeWindow extends Window {
  webkit?: { messageHandlers?: { external?: { postMessage?: unknown } } };
  chrome?: { webview?: { postMessage?: unknown } };
  wails?: { invoke?: unknown };
  _wails?: { environment?: { OS?: string } };
}

export function isDesktop(): boolean {
  if (typeof window === "undefined") return false;
  const native = window as NativeWindow;
  return Boolean(
    native.webkit?.messageHandlers?.external?.postMessage ||
      native.chrome?.webview?.postMessage ||
      native.wails?.invoke ||
      native._wails?.environment?.OS,
  );
}

async function desktop() {
  if (!isDesktop()) throw new Error("Desktop app required");
  return DesktopService;
}

export const api = {
  Profiles: async () => (await desktop()).Profiles(),
  SaveProfile: async (input: ProfileInput) =>
    (await desktop()).SaveProfile(service.ProfileInput.createFrom(input)),
  RemoveProfile: async (id: string, confirmed: boolean) =>
    (await desktop()).RemoveProfile(id, confirmed),
  TestConnection: async (id: string) => (await desktop()).TestConnection(id),
  Buckets: async (id: string) => (await desktop()).Buckets(id),
  List: async (input: ListRequest) => (await desktop()).List(core.ListRequest.createFrom(input)),
  Find: async (input: FindRequest) => (await desktop()).Find(core.FindRequest.createFrom(input)),
  Stat: async (input: ObjectRef) => (await desktop()).Stat(core.ObjectRef.createFrom(input)),
  ReadText: async (input: ObjectRef) =>
    (await desktop()).ReadText(core.ObjectRef.createFrom(input)),
  Share: async (input: ObjectRef, seconds: number, publicLink: boolean) =>
    (await desktop()).Share(core.ObjectRef.createFrom(input), seconds, publicLink),
  Usage: async (id: string, bucket: string, prefix: string, requestID: string) =>
    (await desktop()).Usage(id, bucket, prefix, requestID),
  CancelRequest: async (id: string) => (await desktop()).CancelRequest(id),
  StartTransfer: async (input: TransferRequest) =>
    (await desktop()).StartTransfer(core.TransferRequest.createFrom(input)),
  UploadPaths: async (id: string, bucket: string, prefix: string, paths: string[]) =>
    (await desktop()).UploadPaths(id, bucket, prefix, paths),
  Tasks: async () => (await desktop()).Tasks(),
  ResumeTransfer: async (id: string) => (await desktop()).ResumeTransfer(id),
  PauseTransfer: async (id: string) => (await desktop()).PauseTransfer(id),
  CancelTransfer: async (id: string) => (await desktop()).CancelTransfer(id),
  SetRate: async (bytesPerSecond: number) => (await desktop()).SetRate(bytesPerSecond),
  CreatePlan: async (input: PlanRequest) =>
    (await desktop()).CreatePlan(core.PlanRequest.createFrom(input)),
  ExecutePlan: async (id: string, confirmed: boolean) =>
    (await desktop()).ExecutePlan(id, confirmed),
  GetPlan: async (id: string) => (await desktop()).GetPlan(id),
  PickFiles: async (directory: boolean) => (await desktop()).PickFiles(directory),
  PickDownload: async (name: string) => (await desktop()).PickDownload(name),
  CopyText: async (text: string) => (await desktop()).CopyText(text),
};

export async function onFilesDropped(callback: (paths: string[]) => void): Promise<() => void> {
  if (!isDesktop()) return () => {};
  return Events.On("files:dropped", (event) => {
    const data: unknown = event.data;
    const payload =
      typeof data === "object" && data !== null && "paths" in data ? data.paths : data;
    const paths = Array.isArray(payload)
      ? payload.flat().filter((p): p is string => typeof p === "string")
      : [];
    if (paths.length > 0) callback(paths);
  });
}

export function errorMessage(error: unknown): string {
  if (error instanceof Error) return error.message;
  if (typeof error === "object" && error !== null && "message" in error)
    return String(error.message);
  return String(error);
}
