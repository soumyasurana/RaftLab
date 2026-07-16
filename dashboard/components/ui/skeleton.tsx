import type * as React from "react";
import { cn } from "@/lib/utils";

export function Skeleton({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        "animate-shimmer rounded-2xl bg-gradient-to-r from-slate-800 via-slate-700 to-slate-800 bg-[length:200%_100%]",
        className,
      )}
      {...props}
    />
  );
}
