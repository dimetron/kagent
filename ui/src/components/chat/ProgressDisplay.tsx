import { Loader2 } from "lucide-react";
import { Progress } from "@/components/ui/progress";

interface ProgressDisplayProps {
  message: string;
  progress?: number;
  total?: number;
  percentage?: number;
}

export default function ProgressDisplay({ message, progress, total, percentage }: ProgressDisplayProps) {
  // Calculate percentage if not provided
  const displayPercentage = percentage ?? (progress != null && total != null && total > 0 
    ? (progress / total) * 100 
    : undefined);

  return (
    <div className="flex items-center gap-2 py-2 px-4 border-l-2 border-l-violet-500">
      <div className="flex flex-col gap-2 w-full">
        <div className="flex items-center gap-2">
          <Loader2 className="h-4 w-4 animate-spin text-violet-500" />
          <span className="text-sm text-primary-foreground">{message}</span>
          {displayPercentage != null && (
            <span className="text-xs text-muted-foreground ml-auto">
              {displayPercentage.toFixed(1)}%
            </span>
          )}
        </div>
        {displayPercentage != null && (
          <Progress value={displayPercentage} className="h-2" />
        )}
        {progress != null && total != null && (
          <div className="flex justify-between text-xs text-muted-foreground">
            <span>{progress.toFixed(0)} / {total.toFixed(0)}</span>
            {displayPercentage != null && (
              <span>{(total - progress).toFixed(0)} remaining</span>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

