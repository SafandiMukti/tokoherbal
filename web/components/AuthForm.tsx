"use client";

import { useActionState } from "react";
import type { FormState } from "@/app/actions/auth";

type Props = {
  action: (prev: FormState, form: FormData) => Promise<FormState>;
  mode: "login" | "register";
  next?: string;
};

export function AuthForm({ action, mode, next }: Props) {
  const [state, formAction, pending] = useActionState(action, {} as FormState);
  const isRegister = mode === "register";

  return (
    <form action={formAction} className="auth-form">
      {next && <input type="hidden" name="next" value={next} />}
      <label>
        Email
        <input type="email" name="email" defaultValue={state.email} required autoComplete="email" />
      </label>
      <label>
        Password
        <input
          type="password"
          name="password"
          required
          minLength={isRegister ? 8 : undefined}
          autoComplete={isRegister ? "new-password" : "current-password"}
        />
      </label>
      {isRegister && (
        <label>
          Ulangi password
          <input type="password" name="confirm" required autoComplete="new-password" />
        </label>
      )}
      {state.error && (
        <p className="notice error" role="alert">
          {state.error}
        </p>
      )}
      <button type="submit" disabled={pending}>
        {pending ? "Memproses…" : isRegister ? "Daftar" : "Masuk"}
      </button>
    </form>
  );
}
