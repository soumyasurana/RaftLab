import {
  getConfiguredNodeUrls,
  isClusterChaosAction,
  isDashboardAction,
  issueClusterChaosAction,
  issueNodeAction,
} from "@/lib/management";

export const dynamic = "force-dynamic";

type ActionRequest = {
  nodeUrl?: string;
  action?: unknown;
  payload?: unknown;
};

export async function POST(request: Request) {
  let body: ActionRequest;
  try {
    body = (await request.json()) as ActionRequest;
  } catch {
    return Response.json({ success: false, message: "Invalid JSON request body." }, { status: 400 });
  }

  if (!isDashboardAction(body.action)) {
    return Response.json({ success: false, message: "Unsupported dashboard action." }, { status: 400 });
  }

  if (body.payload !== undefined && (typeof body.payload !== "object" || body.payload === null || Array.isArray(body.payload))) {
    return Response.json({ success: false, message: "Action payload must be an object." }, { status: 400 });
  }

  const nodeUrls = getConfiguredNodeUrls();
  const fallbackNode = nodeUrls[0];
  const nodeUrl = body.nodeUrl ?? fallbackNode;

  if (!nodeUrl) {
    return Response.json(
      { success: false, message: "No management API URLs configured." },
      { status: 400 },
    );
  }

  if (body.nodeUrl && !nodeUrls.includes(body.nodeUrl)) {
    return Response.json({ success: false, message: "Target is not a configured management API." }, { status: 400 });
  }

  const payload = (body.payload ?? {}) as Record<string, unknown>;
  try {
    const result = isClusterChaosAction(body.action)
      ? await issueClusterChaosAction(body.action, payload)
      : await issueNodeAction(nodeUrl, body.action, payload);

    return Response.json(result, { status: result.success ? 200 : 502 });
  } catch {
    return Response.json({ success: false, message: "Management API action failed." }, { status: 502 });
  }
}
