"use client";

import { useState } from "react";
import { Icon } from "./Icon";

// Pemilih jumlah dengan tombol kurang dan tambah. Tetap berupa <input name="qty">
// di dalam form, jadi dikirim seperti isian biasa.
export function QtyInput({
  name = "qty",
  defaultValue = 1,
  min = 1,
  max = 99,
  label,
}: {
  name?: string;
  defaultValue?: number;
  min?: number;
  max?: number;
  label: string;
}) {
  const [v, setV] = useState(String(defaultValue));
  const clamp = (n: number) => Math.min(max, Math.max(min, Number.isFinite(n) ? n : min));
  const num = parseInt(v, 10);

  return (
    <div className="qty">
      <button type="button" className="secondary" aria-label="Kurangi jumlah" disabled={num <= min} onClick={() => setV(String(clamp((num || min) - 1)))}>
        <Icon name="minus" size={16} />
      </button>
      <input
        name={name}
        type="text"
        inputMode="numeric"
        pattern="[0-9]*"
        aria-label={label}
        value={v}
        onChange={(e) => setV(e.target.value.replace(/[^0-9]/g, ""))}
        onBlur={() => setV(String(clamp(num)))}
        required
      />
      <button type="button" className="secondary" aria-label="Tambah jumlah" disabled={num >= max} onClick={() => setV(String(clamp((num || min) + 1)))}>
        <Icon name="plus" size={16} />
      </button>
    </div>
  );
}
