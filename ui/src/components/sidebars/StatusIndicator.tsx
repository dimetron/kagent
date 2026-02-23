import { cn } from "@/lib/utils";

interface StatusIndicatorProps {
  status?: "ok" | "degraded" | "outage";
}

const STATUS_CONFIG = {
  ok:       { dot: "bg-green-500",  label: "All systems operational" },
  degraded: { dot: "bg-yellow-500", label: "Degraded performance" },
  outage:   { dot: "bg-red-500",    label: "Service disruption" },
};

export function StatusIndicator({ status = "ok" }: StatusIndicatorProps) {
  const { dot, label } = STATUS_CONFIG[status];
  return (
    <div className="flex items-center gap-2 px-2 py-1 text-xs text-muted-foreground">
      <span className={cn("h-2 w-2 rounded-full flex-shrink-0", dot)} aria-hidden="true" />
      <span>{label}</span>
    </div>
  );
}
