import { Skeleton } from "@/components/ui/skeleton";

export default function Loading() {
  return (
    <div className="min-h-screen space-y-6 p-6">
      <Skeleton className="h-24 rounded-[2rem]" />
      <div className="grid gap-4 xl:grid-cols-3">
        <Skeleton className="h-28 rounded-[1.75rem]" />
        <Skeleton className="h-28 rounded-[1.75rem]" />
        <Skeleton className="h-28 rounded-[1.75rem]" />
      </div>
      <Skeleton className="h-[34rem] rounded-[2rem]" />
      <Skeleton className="h-[34rem] rounded-[2rem]" />
    </div>
  );
}
