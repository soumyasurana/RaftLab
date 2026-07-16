import { issueNodeAction, getConfiguredNodeUrls } from "@/lib/management";

export const dynamic = "force-dynamic";

type ActionRequest = {
  nodeUrl?: string;
  action:
    | "enableChaos"
    | "disableChaos"
    | "resetChaos"
    | "takeSnapshot"
    | "setLatency"
    | "setPacketLoss"
    | "setPartition"
    | "crashNode"
    | "restartNode"
    | "disconnectNode"
    | "reconnectNode";
  payload?: Record<string, unknown>;
};

export async function POST(request: Request) {
  const body = (await request.json()) as ActionRequest;
  const fallbackNode = getConfiguredNodeUrls()[0];
  const nodeUrl = body.nodeUrl ?? fallbackNode;

  if (!nodeUrl) {
    return Response.json(
      { success: false, message: "No management API URLs configured." },
      { status: 400 },
    );
  }

  const result = await issueNodeAction(nodeUrl, body.action, body.payload);

  return Response.json(result);
}
