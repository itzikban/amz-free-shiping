import { NextRequest, NextResponse } from "next/server";

const BACKEND_BASE_URL = process.env.BACKEND_BASE_URL || "http://127.0.0.1:8085";

export async function GET(req: NextRequest) {
  const url = req.nextUrl.searchParams.get("url");
  const country = req.nextUrl.searchParams.get("country") || "US";
  const zip = req.nextUrl.searchParams.get("zip") || "";

  if (!url) {
    return NextResponse.json({ error: "missing url" }, { status: 400 });
  }

  const backendUrl = new URL("/check", BACKEND_BASE_URL);
  backendUrl.searchParams.set("url", url);
  backendUrl.searchParams.set("country", country);
  if (zip) backendUrl.searchParams.set("zip", zip);

  try {
    const res = await fetch(backendUrl, { method: "GET", cache: "no-store" });
    const body = await res.json();
    // Strip internal details before sending to client
    delete body.method;
    if (body.signal) {
      body.signal = body.signal.replace(/decodo/gi, "proxy");
    }
    return NextResponse.json(body, { status: res.status });
  } catch (err) {
    return NextResponse.json(
      { error: "backend_unreachable", detail: err instanceof Error ? err.message : "unknown" },
      { status: 502 }
    );
  }
}
