"use client";

import { Wifi, WifiOff, AlertCircle, CheckCircle2 } from "lucide-react";
import { cn } from "@/lib/utils";

export type ConnectionHealth = 'healthy' | 'degraded' | 'disconnected' | 'reconnecting';

interface ConnectionStatusProps {
  health: ConnectionHealth;
  className?: string;
  showLabel?: boolean;
  retryCount?: number;
  nextRetryDelay?: string;
}

export function ConnectionStatus({ 
  health, 
  className,
  showLabel = true,
  retryCount = 0,
  nextRetryDelay
}: ConnectionStatusProps) {
  const getStatusInfo = () => {
    switch (health) {
      case 'healthy':
        return {
          icon: CheckCircle2,
          label: 'Connected',
          color: 'text-green-500',
          bgColor: 'bg-green-500/10',
        };
      case 'degraded':
        return {
          icon: AlertCircle,
          label: 'Connection slow',
          color: 'text-yellow-500',
          bgColor: 'bg-yellow-500/10',
        };
      case 'disconnected':
        return {
          icon: WifiOff,
          label: 'Disconnected',
          color: 'text-red-500',
          bgColor: 'bg-red-500/10',
        };
      case 'reconnecting':
        return {
          icon: Wifi,
          label: retryCount > 0 
            ? `Reconnecting (${retryCount})${nextRetryDelay ? ` in ${nextRetryDelay}` : '...'}`
            : 'Reconnecting...',
          color: 'text-blue-500',
          bgColor: 'bg-blue-500/10',
        };
      default:
        return {
          icon: Wifi,
          label: 'Unknown',
          color: 'text-gray-500',
          bgColor: 'bg-gray-500/10',
        };
    }
  };

  const status = getStatusInfo();
  const Icon = status.icon;

  return (
    <div className={cn(
      "flex items-center gap-2 px-2 py-1 rounded-md",
      status.bgColor,
      className
    )}>
      <Icon className={cn("h-4 w-4", status.color, health === 'reconnecting' && "animate-pulse")} />
      {showLabel && (
        <span className={cn("text-xs font-medium", status.color)}>
          {status.label}
        </span>
      )}
    </div>
  );
}

