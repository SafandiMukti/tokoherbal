import Link from "next/link";
import { redirect } from "next/navigation";
import { loginAction } from "@/app/actions/auth";
import { AuthForm } from "@/components/AuthForm";
import { getSession } from "@/lib/authed";
import { safeNext } from "@/lib/session";

export default async function LoginPage({ searchParams }: { searchParams: Promise<{ next?: string }> }) {
  const { next } = await searchParams;
  const target = safeNext(next);
  if (await getSession()) redirect(target);
  return (
    <section className="narrow">
      <h1>Masuk</h1>
      <AuthForm action={loginAction} mode="login" next={target} />
      <p className="muted">
        Belum punya akun? <Link href="/register">Daftar</Link>
      </p>
    </section>
  );
}
