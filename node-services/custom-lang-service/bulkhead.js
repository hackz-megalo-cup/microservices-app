import pLimit from "p-limit";

export function createBulkhead(maxConcurrent) {
  const limit = pLimit(maxConcurrent);
  return {
    execute: (fn) => limit(fn),
    pendingCount: () => limit.pendingCount,
    activeCount: () => limit.activeCount,
  };
}
