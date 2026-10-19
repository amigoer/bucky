import { create } from "zustand";
import type { OperationPlan } from "./api";
import type { Entry } from "./entries";

export interface Location {
  profileId: string;
  bucket: string;
  prefix: string;
}

export type Page = "browse" | "transfers" | "usage" | "settings";
export type PlanOperation = "delete" | "copy" | "move" | "upload";

export interface PlanDraft {
  operation: PlanOperation;
  /** Where the selection lives; copy and move keep paths below its prefix. */
  location: Location;
  entries: Entry[];
  plan?: OperationPlan;
}

export interface ShareDraft {
  location: Location;
  entries: Entry[];
}

export const sameLocation = (a: Location | null, b: Location | null) =>
  !!a && !!b && a.profileId === b.profileId && a.bucket === b.bucket && a.prefix === b.prefix;

interface NavState {
  location: Location | null;
  backStack: Location[];
  forwardStack: Location[];
  page: Page;
  selection: string[];
  anchor: string | null;
  query: string;
  quickLook: string | null;
  /** Siblings the preview can step through; null falls back to the folder listing. */
  quickLookList: Entry[] | null;
  share: ShareDraft | null;
  plan: PlanDraft | null;
  planQueue: PlanDraft[];
  connection: { id?: string } | null;
  quickUpload: { file?: File } | null;
  searchOpen: boolean;

  navigate: (location: Location) => void;
  goBack: () => void;
  goForward: () => void;
  setPage: (page: Page) => void;
  select: (keys: string[], anchor?: string | null) => void;
  setQuery: (query: string) => void;
  setQuickLook: (key: string | null, list?: Entry[] | null) => void;
  openShare: (draft: ShareDraft | null) => void;
  openPlan: (draft: PlanDraft) => void;
  queuePlans: (drafts: PlanDraft[]) => void;
  closePlan: () => void;
  openConnection: (value: { id?: string } | null) => void;
  openQuickUpload: (value: { file?: File } | null) => void;
  setSearchOpen: (open: boolean) => void;
  forgetProfile: (profileId: string) => void;
}

export const useNav = create<NavState>()((set, get) => ({
  location: null,
  backStack: [],
  forwardStack: [],
  page: "browse",
  selection: [],
  anchor: null,
  query: "",
  quickLook: null,
  quickLookList: null,
  share: null,
  plan: null,
  planQueue: [],
  connection: null,
  quickUpload: null,
  searchOpen: false,

  navigate: (location) => {
    const { location: current, backStack } = get();
    if (sameLocation(current, location)) {
      set({ page: "browse" });
      return;
    }
    set({
      location,
      backStack: current ? [...backStack.slice(-49), current] : backStack,
      forwardStack: [],
      page: "browse",
      selection: [],
      anchor: null,
      query: "",
      quickLook: null,
      quickLookList: null,
    });
  },
  goBack: () => {
    const { backStack, location, forwardStack } = get();
    const target = backStack.at(-1);
    if (!target) return;
    set({
      location: target,
      backStack: backStack.slice(0, -1),
      forwardStack: location ? [...forwardStack, location] : forwardStack,
      selection: [],
      anchor: null,
      query: "",
      page: "browse",
    });
  },
  goForward: () => {
    const { backStack, location, forwardStack } = get();
    const target = forwardStack.at(-1);
    if (!target) return;
    set({
      location: target,
      forwardStack: forwardStack.slice(0, -1),
      backStack: location ? [...backStack, location] : backStack,
      selection: [],
      anchor: null,
      query: "",
      page: "browse",
    });
  },
  setPage: (page) => set({ page, quickLook: null }),
  select: (keys, anchor) =>
    set((s) => ({ selection: keys, anchor: anchor === undefined ? s.anchor : anchor })),
  setQuery: (query) => set({ query, selection: [], anchor: null }),
  setQuickLook: (quickLook, list) =>
    set({ quickLook, quickLookList: quickLook ? (list ?? null) : null }),
  openShare: (share) => set({ share, quickLook: null, quickLookList: null }),
  openPlan: (plan) => set({ plan, quickLook: null, quickLookList: null }),
  queuePlans: (drafts) => {
    if (drafts.length === 0) return;
    const { plan, planQueue } = get();
    if (plan) set({ planQueue: [...planQueue, ...drafts] });
    else set({ plan: drafts[0], planQueue: [...planQueue, ...drafts.slice(1)] });
  },
  closePlan: () => {
    const [next, ...rest] = get().planQueue;
    set({ plan: next ?? null, planQueue: rest });
  },
  openConnection: (connection) => set({ connection }),
  openQuickUpload: (quickUpload) => set({ quickUpload, quickLook: null, quickLookList: null }),
  setSearchOpen: (searchOpen) => set({ searchOpen }),
  forgetProfile: (profileId) => {
    const keep = (l: Location) => l.profileId !== profileId;
    const { location, backStack, forwardStack } = get();
    set({
      location: location && keep(location) ? location : null,
      backStack: backStack.filter(keep),
      forwardStack: forwardStack.filter(keep),
      selection: [],
    });
  },
}));
