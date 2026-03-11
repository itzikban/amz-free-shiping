import { NextResponse } from "next/server";

const ADMIN_USERNAME = process.env.LOCAL_ADMIN_USERNAME || "itzik_admin";
const ADMIN_PASSWORD = process.env.LOCAL_ADMIN_PASSWORD || "Itzik@2026!Local#9";
const ADMIN_API_TOKEN = process.env.ADMIN_API_TOKEN;

export async function POST(req: Request) {
  if (!ADMIN_API_TOKEN || !ADMIN_API_TOKEN.trim()) {
    return NextResponse.json({ error: "admin_token_not_configured" }, { status: 500 });
  }

  const body = await req.json().catch(() => ({}));
  const username = typeof body?.username === "string" ? body.username.trim() : "";
  const password = typeof body?.password === "string" ? body.password : "";

  if (username !== ADMIN_USERNAME || password !== ADMIN_PASSWORD) {
    return NextResponse.json({ error: "invalid_credentials" }, { status: 401 });
  }

  const res = NextResponse.json({ ok: true });
  res.cookies.set("admin_session", ADMIN_API_TOKEN, {
    httpOnly: true,
    sameSite: "lax",
    secure: false,
    path: "/",
    maxAge: 60 * 60 * 8,
  });
  return res;
}
