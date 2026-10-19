// Static imports avoid a cyclic chunk graph that WebKit evaluates out of order.
import { Events } from "@wailsio/runtime";
import { DesktopService } from "../../bindings/github.com/amigoer/bucky/internal/app/index";
import * as app from "../../bindings/github.com/amigoer/bucky/internal/app/models";
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
export type PlanItem = core.PlanItem;
export type ItemResult = core.ItemResult;
export type TransferTask = core.TransferTask;
export type TransferRequest = core.TransferRequest;
export type ShareLink = core.ShareLink;
export type TextPreview = core.TextPreview;
export type Usage = core.Usage;
export type FindRequest = core.FindRequest;
export type FindResult = core.FindResult;
export type ProfileInput = service.ProfileInput;
export type TransferResult = service.TransferResult;
export type DownloadResult = app.DownloadResult;
export type UsageProgress = app.UsageProgress;

export const models = { core, service, app };

/** Every desktop method the renderer uses; the dev mock implements the same surface. */
export interface Backend {
  Profiles(): Promise<Profile[]>;
  SaveProfile(input: ProfileInput): Promise<Profile>;
  RemoveProfile(id: string, confirmed: boolean): Promise<void>;
  TestConnection(id: string): Promise<Bucket[]>;
  TestDraft(input: ProfileInput): Promise<Bucket[]>;
  Buckets(id: string): Promise<Bucket[]>;
  List(req: ListRequest): Promise<ListPage>;
  Find(req: FindRequest, signal?: AbortSignal): Promise<FindResult>;
  Stat(ref: ObjectRef): Promise<ObjectInfo>;
  ReadText(ref: ObjectRef): Promise<TextPreview>;
  Share(ref: ObjectRef, seconds: number, publicLink: boolean): Promise<ShareLink>;
  Usage(id: string, bucket: string, prefix: string, requestID: string): Promise<Usage>;
  CancelRequest(id: string): Promise<void>;
  StartTransfer(req: TransferRequest): Promise<TransferResult>;
  UploadPaths(
    id: string,
    bucket: string,
    prefix: string,
    paths: string[],
  ): Promise<TransferResult[]>;
  UploadData(ref: ObjectRef, base64: string): Promise<TransferResult>;
  DownloadTo(dir: string, sources: ObjectRef[], base: string): Promise<DownloadResult[]>;
  Tasks(): Promise<TransferTask[]>;
  ResumeTransfer(id: string): Promise<TransferTask>;
  PauseTransfer(id: string): Promise<void>;
  CancelTransfer(id: string): Promise<void>;
  SetRate(bytesPerSecond: number): Promise<void>;
  SetConcurrency(n: number): Promise<number>;
  CreatePlan(req: PlanRequest): Promise<OperationPlan>;
  ExecutePlan(id: string, confirmed: boolean): Promise<OperationPlan>;
  GetPlan(id: string): Promise<OperationPlan>;
  PickFiles(directory: boolean): Promise<string[]>;
  PickDirectory(): Promise<string>;
  PickDownload(name: string): Promise<string>;
  CopyText(text: string): Promise<void>;
}

export interface EventSource {
  onTransferUpdated(callback: (task: TransferTask) => void): () => void;
  onUsageProgress(callback: (progress: UsageProgress) => void): () => void;
  onFilesDropped(callback: (paths: string[]) => void): () => void;
}

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

const desktop: Backend = {
  Profiles: () => DesktopService.Profiles(),
  SaveProfile: (input) => DesktopService.SaveProfile(service.ProfileInput.createFrom(input)),
  RemoveProfile: (id, confirmed) => DesktopService.RemoveProfile(id, confirmed),
  TestConnection: (id) => DesktopService.TestConnection(id),
  TestDraft: (input) => DesktopService.TestDraft(service.ProfileInput.createFrom(input)),
  Buckets: (id) => DesktopService.Buckets(id),
  List: (req) => DesktopService.List(core.ListRequest.createFrom(req)),
  Find: (req, signal) => {
    const call = DesktopService.Find(core.FindRequest.createFrom(req));
    // Cancelling the call cancels the scan's context in the Go service.
    signal?.addEventListener("abort", () => void call.cancel());
    return call;
  },
  Stat: (ref) => DesktopService.Stat(core.ObjectRef.createFrom(ref)),
  ReadText: (ref) => DesktopService.ReadText(core.ObjectRef.createFrom(ref)),
  Share: (ref, seconds, publicLink) =>
    DesktopService.Share(core.ObjectRef.createFrom(ref), seconds, publicLink),
  Usage: (id, bucket, prefix, requestID) => DesktopService.Usage(id, bucket, prefix, requestID),
  CancelRequest: (id) => DesktopService.CancelRequest(id),
  StartTransfer: (req) => DesktopService.StartTransfer(core.TransferRequest.createFrom(req)),
  UploadPaths: (id, bucket, prefix, paths) => DesktopService.UploadPaths(id, bucket, prefix, paths),
  UploadData: (ref, base64) => DesktopService.UploadData(core.ObjectRef.createFrom(ref), base64),
  DownloadTo: (dir, sources, base) =>
    DesktopService.DownloadTo(
      dir,
      sources.map((s) => core.ObjectRef.createFrom(s)),
      base,
    ),
  Tasks: () => DesktopService.Tasks(),
  ResumeTransfer: (id) => DesktopService.ResumeTransfer(id),
  PauseTransfer: (id) => DesktopService.PauseTransfer(id),
  CancelTransfer: (id) => DesktopService.CancelTransfer(id),
  SetRate: (bytesPerSecond) => DesktopService.SetRate(bytesPerSecond),
  SetConcurrency: (n) => DesktopService.SetConcurrency(n),
  CreatePlan: (req) => DesktopService.CreatePlan(core.PlanRequest.createFrom(req)),
  ExecutePlan: (id, confirmed) => DesktopService.ExecutePlan(id, confirmed),
  GetPlan: (id) => DesktopService.GetPlan(id),
  PickFiles: (directory) => DesktopService.PickFiles(directory),
  PickDirectory: () => DesktopService.PickDirectory(),
  PickDownload: (name) => DesktopService.PickDownload(name),
  CopyText: (text) => DesktopService.CopyText(text),
};

function payloadPaths(data: unknown): string[] {
  const payload =
    typeof data === "object" && data !== null && "paths" in data
      ? (data as { paths: unknown }).paths
      : data;
  return Array.isArray(payload)
    ? payload.flat().filter((p): p is string => typeof p === "string")
    : [];
}

const desktopEvents: EventSource = {
  onTransferUpdated: (callback) =>
    Events.On("transfer:updated", (event) => callback(core.TransferTask.createFrom(event.data))),
  onUsageProgress: (callback) =>
    Events.On("usage:progress", (event) => callback(app.UsageProgress.createFrom(event.data))),
  onFilesDropped: (callback) =>
    Events.On("files:dropped", (event) => {
      const paths = payloadPaths(event.data);
      if (paths.length > 0) callback(paths);
    }),
};

const unavailable = (): Promise<never> =>
  Promise.reject({ message: "Desktop app required", cause: { code: "desktop_required" } });

const offline = new Proxy({} as Backend, { get: () => unavailable });

const noEvents: EventSource = {
  onTransferUpdated: () => () => {},
  onUsageProgress: () => () => {},
  onFilesDropped: () => () => {},
};

export type BackendKind = "desktop" | "mock" | "none";

let current: Backend = isDesktop() ? desktop : offline;
let currentEvents: EventSource = isDesktop() ? desktopEvents : noEvents;
let kind: BackendKind = isDesktop() ? "desktop" : "none";

/** Resolves once the backend is chosen; the mock only exists in dev builds. */
export const backendReady: Promise<void> = (async () => {
  if (import.meta.env.DEV && !isDesktop()) {
    const mock = await import("./mock");
    const created = mock.createMockBackend();
    current = created.backend;
    currentEvents = created.events;
    kind = "mock";
  }
})();

export function backendKind(): BackendKind {
  return kind;
}

export const api: Backend = new Proxy({} as Backend, {
  get:
    (_target, name: keyof Backend) =>
    (...args: unknown[]) =>
      (current[name] as (...a: unknown[]) => Promise<unknown>)(...args),
});

export const events: EventSource = {
  onTransferUpdated: (cb) => currentEvents.onTransferUpdated(cb),
  onUsageProgress: (cb) => currentEvents.onUsageProgress(cb),
  onFilesDropped: (cb) => currentEvents.onFilesDropped(cb),
};

export const ref = (profileId: string, bucket: string, key: string): ObjectRef =>
  core.ObjectRef.createFrom({ profileId, bucket, key });
