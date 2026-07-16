"use client";

import { useEffect } from "react";
import { AlertTriangle, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <div className="flex min-h-screen items-center justify-center p-6">
      <div className="max-w-lg rounded-[2rem] border border-white/8 bg-white/[0.04] p-8 text-center shadow-glow">
        <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl border border-rose-400/20 bg-rose-500/10 text-rose-200">
          <AlertTriangle className="h-6 w-6" />
        </div>
        <h2 className="mt-5 text-2xl font-semibold text-white">Dashboard failed to load</h2>
        <p className="mt-3 text-sm leading-6 text-slate-400">
          Something in the live cluster view threw an exception. You can retry the render without reloading the page.
        </p>
        <Button className="mt-6" onClick={reset}>
          <RefreshCw className="h-4 w-4" />
          Retry
        </Button>
      </div>
    </div>
  );
}
