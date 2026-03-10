import { NextRequest, NextResponse } from "next/server";

const BACKEND_BASE_URL = process.env.BACKEND_BASE_URL || "http://127.0.0.1:8085";

export async function POST(req: NextRequest) {
  const id = req.nextUrl.searchParams.get("id");
  if (!id) return NextResponse.json({ error: "missing id" }, { status: 400 });
  try {
    const res = await fetch(new URL(`/monitor/stop?id=${encodeURIComponent(id)}`, BACKEND_BASE_URL), { method: "POST", cache: "no-store" });
    return NextResponse.json(await res.json(), { status: res.status });
  } catch (err) {
    return NextResponse.json({ error: "backend_unreachable", detail: err instanceof Error ? err.message : "unknown" }, { status: 502 });
  }
}
