"use client";

import * as React from "react";
import { cn } from "@/lib/utils";

type ButtonVariant = "default" | "secondary" | "outline" | "ghost" | "destructive";

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: "sm" | "default" | "lg";
}

const variantClasses: Record<ButtonVariant, string> = {
  default:
    "bg-emerald-400 text-slate-950 hover:bg-emerald-300 shadow-[0_0_0_1px_rgba(16,185,129,0.15)]",
  secondary: "bg-slate-800 text-slate-100 hover:bg-slate-700",
  outline: "border border-slate-700 bg-transparent text-slate-100 hover:bg-slate-800",
  ghost: "bg-transparent text-slate-100 hover:bg-slate-800",
  destructive: "bg-rose-500 text-white hover:bg-rose-400",
};

const sizeClasses = {
  sm: "h-8 px-3 text-xs",
  default: "h-10 px-4 text-sm",
  lg: "h-12 px-5 text-sm",
};

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant = "default", size = "default", type = "button", ...props }, ref) => {
    return (
      <button
        ref={ref}
        type={type}
        className={cn(
          "inline-flex items-center justify-center gap-2 rounded-full font-medium transition-all duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-400/60 disabled:pointer-events-none disabled:opacity-50",
          variantClasses[variant],
          sizeClasses[size],
          className,
        )}
        {...props}
      />
    );
  },
);

Button.displayName = "Button";
