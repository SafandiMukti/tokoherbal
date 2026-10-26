import Link from "next/link";
import { redirect } from "next/navigation";
import { registerAction } from "@/app/actions/auth";
import { AuthShell } from "@/components/AuthShell";
import { AuthForm } from "@/components/AuthForm";
import { getSession } from "@/lib/authed";

export default async function RegisterPage() {
  if (await getSession()) redirect("/");
  return (
    <AuthShell title="Daftar">
      <AuthForm action={registerAction} mode="register" />
      <p className="muted">
        Sudah punya akun? <Link href="/login">Masuk</Link>
      </p>
    </AuthShell>
  );
}
