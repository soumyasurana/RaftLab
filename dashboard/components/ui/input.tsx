import * as React from "react";
import { cn } from "@/lib/utils";

export const Input = React.forwardRef<
  HTMLInputElement,
  React.InputHTMLAttributes<HTMLInputElement>
>(({ className, ...props }, ref) => {
  return (
    <input
      ref={ref}
      className={cn(
        "flex h-10 w-full rounded-xl border border-white/10 bg-slate-900/80 px-3 text-sm text-slate-100 outline-none transition-colors placeholder:text-slate-500 focus:border-emerald-400/50 focus:ring-1 focus:ring-emerald-400/40",
        className,
      )}
      {...props}
    />
  );
});

Input.displayName = "Input";
