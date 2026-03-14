import { NextRequest, NextResponse } from "next/server";

const BACKEND_BASE_URL = process.env.BACKEND_BASE_URL || "http://127.0.0.1:8085";

/**
 * Handle GET requests by validating query parameters and proxying a '/check' request to the backend.
 *
 * Expects query parameters on the incoming request:
 * - `url` (required)
 * - `country` (optional, defaults to `"US"`)
 * - `zip` (optional)
 * - `method` (optional)
 *
 * @param req - NextRequest containing the query parameters described above
 * @returns A JSON response containing the backend's parsed JSON body with the backend's HTTP status. If `url` is missing, responds with status 400 and `{ error: "missing url" }`. If the backend is unreachable, responds with status 502 and `{ error: "backend_unreachable", detail: string }`.
 */
export async function GET(req: NextRequest) {
  const url = req.nextUrl.searchParams.get("url");
  const country = req.nextUrl.searchParams.get("country") || "US";
  const zip = req.nextUrl.searchParams.get("zip") || "";
  const method = req.nextUrl.searchParams.get("method") || "";

  if (!url) {
    return NextResponse.json({ error: "missing url" }, { status: 400 });
  }

  const backendUrl = new URL("/check", BACKEND_BASE_URL);
  backendUrl.searchParams.set("url", url);
  backendUrl.searchParams.set("country", country);
  if (zip) backendUrl.searchParams.set("zip", zip);
  if (method) backendUrl.searchParams.set("method", method);

  try {
    const res = await fetch(backendUrl, { method: "GET", cache: "no-store" });
    const body = await res.json();
    return NextResponse.json(body, { status: res.status });
  } catch (err) {
    return NextResponse.json(
      { error: "backend_unreachable", detail: err instanceof Error ? err.message : "unknown" },
      { status: 502 }
    );
  }
}
