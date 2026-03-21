import { NextRequest, NextResponse } from "next/server";

const BACKEND_BASE_URL = process.env.BACKEND_BASE_URL || "http://127.0.0.1:8085";

export async function GET(req: NextRequest) {
  const url = req.nextUrl.searchParams.get("url");
  const country = req.nextUrl.searchParams.get("country") || "US";
  const zip = req.nextUrl.searchParams.get("zip") || "";
  const threshold = req.nextUrl.searchParams.get("threshold") || "50";
  const method = req.nextUrl.searchParams.get("method") || "";

  if (!url) {
    return NextResponse.json({ error: "missing url" }, { status: 400 });
  }

  const backendUrl = new URL("/api/v1/fill-to-threshold", BACKEND_BASE_URL);
  backendUrl.searchParams.set("url", url);
  backendUrl.searchParams.set("country", country);
  backendUrl.searchParams.set("threshold", threshold);
  if (zip) backendUrl.searchParams.set("zip", zip);
  if (method) backendUrl.searchParams.set("method", method);

  try {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 10_000);
    let res: Response;
    try {
      res = await fetch(backendUrl, { method: "GET", cache: "no-store", signal: controller.signal });
    } finally {
      clearTimeout(timeoutId);
    }
    const raw = await res.text();

    try {
      const body = JSON.parse(raw);
      return NextResponse.json(body, { status: res.status });
    } catch {
      // Backend returned a non-JSON body (e.g. HTML error page).
      return NextResponse.json(
        { error: "backend_invalid_response" },
        { status: 422 }
      );
    }
  } catch {
    // Backend is unreachable or request timed out.
    return NextResponse.json(
      { error: "backend_unreachable" },
      { status: 502 }
    );
  }
}
