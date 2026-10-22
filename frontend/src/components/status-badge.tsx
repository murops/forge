import { statusLabel } from "@/lib/status";
import type { Status } from "@/lib/types";

const styles: Record<Status, string> = {
  pending: "bg-zinc-500/15 text-zinc-600 dark:text-zinc-300",
  running: "bg-blue-500/15 text-blue-700 dark:text-blue-300",
  succeeded: "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300",
  failed: "bg-red-500/15 text-red-700 dark:text-red-300",
};

export function StatusBadge({ status }: { status: Status }) {
  return (
    <span className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium ${styles[status]}`}>
      {status === "running" && <span className="size-1.5 animate-pulse rounded-full bg-current" />}
      {statusLabel[status]}
    </span>
  );
}
