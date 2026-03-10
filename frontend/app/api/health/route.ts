import { NextResponse } from "next/server";

const BACKEND_BASE_URL = process.env.BACKEND_BASE_URL || "http://127.0.0.1:8085";

export async function GET() {
  try {
    const res = await fetch(new URL("/health", BACKEND_BASE_URL), { cache: "no-store" });
    const body = await res.json();
    return NextResponse.json({ ok: res.ok, backend: body }, { status: res.status });
  } catch (err) {
    return NextResponse.json(
      { ok: false, error: "backend_unreachable", detail: err instanceof Error ? err.message : "unknown" },
      { status: 502 }
    );
  }
}
