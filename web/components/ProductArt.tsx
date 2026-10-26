import { useId } from "react";
import { styleFor } from "@/lib/categories";
import { Vessel } from "./Vessel";

// Gambar produk pengganti: wadah sesuai kategori di atas latar warna dengan pola kawung samar.
export function ProductArt({ category, name }: { category: string; name: string }) {
  const st = styleFor(category);
  const pid = useId().replace(/:/g, "");
  return (
    <div className="art" style={{ background: st.bg }} role="img" aria-label={`Ilustrasi ${name}`}>
      <svg className="art-pattern" aria-hidden="true" focusable="false">
        <defs>
          <pattern id={pid} width="44" height="44" patternUnits="userSpaceOnUse">
            <g fill="none" stroke={st.deep} strokeWidth="1.1">
              <ellipse cx="22" cy="11" rx="9" ry="5" transform="rotate(90 22 11)" />
              <ellipse cx="22" cy="33" rx="9" ry="5" transform="rotate(90 22 33)" />
              <ellipse cx="11" cy="22" rx="9" ry="5" />
              <ellipse cx="33" cy="22" rx="9" ry="5" />
            </g>
          </pattern>
        </defs>
        <rect width="100%" height="100%" fill={`url(#${pid})`} />
      </svg>
      <Vessel category={category} className="art-vessel" />
    </div>
  );
}
