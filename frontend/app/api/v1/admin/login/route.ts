import { timingSafeEqual } from "crypto";
import { NextResponse } from "next/server";

const ADMIN_USERNAME = process.env.LOCAL_ADMIN_USERNAME;
const ADMIN_PASSWORD = process.env.LOCAL_ADMIN_PASSWORD;
const ADMIN_API_TOKEN = process.env.ADMIN_API_TOKEN?.trim();

function safeCompare(a: string, b: string): boolean {
  const aBuf = Buffer.from(a);
  const bBuf = Buffer.from(b);
  const maxLen = Math.max(aBuf.length, bBuf.length, 1);
  const aPadded = Buffer.alloc(maxLen);
  const bPadded = Buffer.alloc(maxLen);
  aBuf.copy(aPadded);
  bBuf.copy(bPadded);
  return timingSafeEqual(aPadded, bPadded) && aBuf.length === bBuf.length;
}

export async function POST(req: Request) {
  if (!ADMIN_API_TOKEN?.trim() || !ADMIN_USERNAME?.trim() || !ADMIN_PASSWORD) {
    console.error("admin login is misconfigured: LOCAL_ADMIN_USERNAME, LOCAL_ADMIN_PASSWORD, and ADMIN_API_TOKEN are required");
    return NextResponse.json({ error: "admin_login_not_configured" }, { status: 500 });
  }

  const body = await req.json().catch(() => ({}));
  const username = typeof body?.username === "string" ? body.username.trim() : "";
  const password = typeof body?.password === "string" ? body.password : "";

  const okUser = safeCompare(username, ADMIN_USERNAME);
  const okPass = safeCompare(password, ADMIN_PASSWORD);
  if (!(okUser && okPass)) {
    return NextResponse.json({ error: "invalid_credentials" }, { status: 401 });
  }

  const isHttps = (() => {
    try {
      const proto = req.headers.get("x-forwarded-proto");
      if (proto) return proto.toLowerCase().includes("https");
      return new URL(req.url).protocol === "https:";
    } catch {
      return false;
    }
  })();

  const res = NextResponse.json({ ok: true });
  res.cookies.set("admin_session", ADMIN_API_TOKEN, {
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production" && isHttps,
    path: "/",
    maxAge: 60 * 60 * 8,
  });
  return res;
}
