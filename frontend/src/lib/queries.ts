import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { useEffect } from "react";
import { api, events, type TransferTask } from "./api";
import type { Location } from "./nav";
import { queryClient } from "./query";

export const PAGE_SIZE = 1000;

export const keys = {
  profiles: ["profiles"] as const,
  buckets: (profileId: string) => ["buckets", profileId] as const,
  list: (l: Location) => ["list", l.profileId, l.bucket, l.prefix] as const,
  listing: (profileId: string, bucket: string) => ["list", profileId, bucket] as const,
  find: (l: Location, query: string) => ["find", l.profileId, l.bucket, l.prefix, query] as const,
  tasks: ["tasks"] as const,
};

export function useProfiles() {
  return useQuery({ queryKey: keys.profiles, queryFn: () => api.Profiles(), staleTime: Infinity });
}

export function useBuckets(profileId: string | undefined, enabled = true) {
  return useQuery({
    queryKey: keys.buckets(profileId ?? ""),
    queryFn: () => api.Buckets(profileId ?? ""),
    enabled: !!profileId && enabled,
    staleTime: 5 * 60_000,
    retry: false,
  });
}

export function useListing(location: Location | null) {
  return useInfiniteQuery({
    queryKey: location ? keys.list(location) : ["list", "none"],
    enabled: !!location,
    initialPageParam: "",
    queryFn: ({ pageParam }) => {
      const l = location as Location;
      return api.List({
        profileId: l.profileId,
        bucket: l.bucket,
        prefix: l.prefix,
        delimiter: "/",
        cursor: pageParam,
        limit: PAGE_SIZE,
      });
    },
    getNextPageParam: (page) => page.nextCursor || undefined,
    retry: false,
  });
}

export function useSearch(location: Location | null, query: string) {
  const q = query.trim();
  return useInfiniteQuery({
    queryKey: location ? keys.find(location, q) : ["find", "none"],
    enabled: !!location && q.length > 0,
    initialPageParam: "",
    queryFn: ({ pageParam, signal }) => {
      const l = location as Location;
      return api.Find(
        {
          profileId: l.profileId,
          bucket: l.bucket,
          prefix: l.prefix,
          contains: q,
          before: "",
          minSize: 0,
          maxSize: 0,
          limit: PAGE_SIZE,
          cursor: pageParam,
        },
        signal,
      );
    },
    getNextPageParam: (page) => (page.complete ? undefined : page.nextCursor || undefined),
    staleTime: 10_000,
    retry: false,
  });
}

const ACTIVE = new Set(["queued", "running", "cancelling"]);

export function isActiveTask(task: TransferTask) {
  return ACTIVE.has(task.status);
}

export function usePresigned(
  location: Pick<Location, "profileId" | "bucket"> | null,
  key: string,
  etag: string,
  enabled: boolean,
) {
  return useQuery({
    queryKey: ["presign", location?.profileId, location?.bucket, key, etag],
    queryFn: () =>
      api.Share(
        { profileId: location?.profileId ?? "", bucket: location?.bucket ?? "", key },
        3600,
        false,
      ),
    enabled: !!location && enabled,
    // Links are signed for an hour and kept in memory only.
    staleTime: 50 * 60_000,
    gcTime: 55 * 60_000,
    retry: false,
  });
}

export function useTasks() {
  return useQuery({
    queryKey: keys.tasks,
    queryFn: () => api.Tasks(),
    staleTime: 5_000,
    refetchInterval: (query) => (query.state.data?.some(isActiveTask) ? 5_000 : false),
  });
}

/** Merges live transfer events into the task cache and refreshes listings an upload changed. */
export function useTransferEvents() {
  useEffect(() => {
    return events.onTransferUpdated((task) => {
      const previous = queryClient
        .getQueryData<TransferTask[]>(keys.tasks)
        ?.find((t) => t.id === task.id);
      queryClient.setQueryData<TransferTask[]>(keys.tasks, (tasks) => {
        if (!tasks) return [task];
        const index = tasks.findIndex((t) => t.id === task.id);
        if (index < 0) return [task, ...tasks];
        const next = tasks.slice();
        next[index] = task;
        return next;
      });
      if (
        task.direction === "upload" &&
        task.status === "completed" &&
        previous?.status !== "completed"
      ) {
        void queryClient.invalidateQueries({
          queryKey: keys.listing(task.object.profileId, task.object.bucket),
        });
      }
    });
  }, []);
}

export function invalidateBucket(profileId: string, bucket: string) {
  void queryClient.invalidateQueries({ queryKey: keys.listing(profileId, bucket) });
  void queryClient.invalidateQueries({ queryKey: ["find", profileId, bucket] });
}
