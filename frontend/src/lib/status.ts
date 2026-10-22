import type { Status } from "@/lib/types";

export const statusLabel: Record<Status, string> = {
  pending: "Queued",
  running: "Provisioning",
  succeeded: "Ready",
  failed: "Failed",
};

export function isFinished(status: Status) {
  return status === "succeeded" || status === "failed";
}
