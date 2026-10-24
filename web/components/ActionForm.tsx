"use client";

import { useActionState, type ReactNode } from "react";
import type { ShopState } from "@/lib/types";

// Pembungkus form untuk server action: menonaktifkan isian saat diproses dan
// menampilkan pesan galat/sukses dari server.
export function ActionForm({
  action,
  children,
  className,
}: {
  action: (prev: ShopState, form: FormData) => Promise<ShopState>;
  children: ReactNode;
  className?: string;
}) {
  const [state, formAction, pending] = useActionState(action, {} as ShopState);
  return (
    <form action={formAction} className={className} aria-busy={pending}>
      <fieldset disabled={pending} className="bare">
        {children}
      </fieldset>
      {state.error && (
        <p className="notice error" role="alert">
          {state.error}
        </p>
      )}
      {state.ok && (
        <p className="notice ok" role="status">
          {state.ok}
        </p>
      )}
    </form>
  );
}
