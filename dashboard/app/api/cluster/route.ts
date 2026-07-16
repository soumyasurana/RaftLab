import { loadClusterState } from "@/lib/management";

export const dynamic = "force-dynamic";

export async function GET() {
  const cluster = await loadClusterState();

  return Response.json(cluster, {
    headers: {
      "Cache-Control": "no-store, max-age=0",
    },
  });
}
