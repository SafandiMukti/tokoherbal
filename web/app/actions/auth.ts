"use server";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import * as authApi from "@/lib/auth-api";
import {
  ACCESS_COOKIE,
  REFRESH_COOKIE,
  REFRESH_MAX_AGE,
  cookieOptions,
  safeNext,
  type TokenResponse,
} from "@/lib/session";

export type FormState = { error?: string; email?: string };

async function startSession(t: TokenResponse) {
  const store = await cookies();
  store.set(ACCESS_COOKIE, t.access_token, cookieOptions(t.expires_in));
  store.set(REFRESH_COOKIE, t.refresh_token, cookieOptions(REFRESH_MAX_AGE));
}

export async function loginAction(_prev: FormState, form: FormData): Promise<FormState> {
  const email = String(form.get("email") ?? "").trim();
  const password = String(form.get("password") ?? "");
  const next = safeNext(String(form.get("next") ?? ""));
  if (!email || !password) return { error: "Email dan password wajib diisi.", email };

  const r = await authApi.login(email, password);
  if (!r.ok) return { error: r.message, email };

  await startSession(r.data);
  redirect(next); // redirect() melempar error khusus; jangan dibungkus try/catch
}

export async function registerAction(_prev: FormState, form: FormData): Promise<FormState> {
  const email = String(form.get("email") ?? "").trim();
  const password = String(form.get("password") ?? "");
  const confirm = String(form.get("confirm") ?? "");
  if (!email || !password) return { error: "Email dan password wajib diisi.", email };
  if (password !== confirm) return { error: "Konfirmasi password tidak sama.", email };

  const reg = await authApi.register(email, password);
  if (!reg.ok) return { error: reg.message, email };

  // langsung masuk setelah mendaftar
  const r = await authApi.login(email, password);
  if (!r.ok) redirect("/login");
  await startSession(r.data);
  redirect("/");
}

export async function logoutAction() {
  const store = await cookies();
  const refreshToken = store.get(REFRESH_COOKIE)?.value;
  // cabut di server dulu; bila auth-service mati, cookie tetap dihapus di browser ini
  if (refreshToken) await authApi.logout(refreshToken);
  store.delete(ACCESS_COOKIE);
  store.delete(REFRESH_COOKIE);
  redirect("/");
}
