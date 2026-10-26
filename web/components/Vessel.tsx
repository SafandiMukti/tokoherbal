import { useId } from "react";
import { styleFor, type CategoryStyle } from "@/lib/categories";

// Wadah (botol, toples, dst.) digambar di kotak 200 x 250. Dipakai di hero (tanpa latar)
// dan sebagai gambar produk bila admin belum mengunggah foto.
const GLASS = "#FFFFFF";

function Label({ x, y, w, h, st }: { x: number; y: number; w: number; h: number; st: CategoryStyle }) {
  // label kertas dengan tanda daun kecil
  const cx = x + w / 2;
  const cy = y + h / 2 - 3;
  return (
    <g>
      <rect x={x} y={y} width={w} height={h} rx={4} fill="#FFFFFF" opacity={0.94} />
      <path d={`M${cx - 7} ${cy + 8} C${cx - 7} ${cy - 4} ${cx + 1} ${cy - 10} ${cx + 9} ${cy - 10} C${cx + 9} ${cy} ${cx + 3} ${cy + 8} ${cx - 7} ${cy + 8}Z`} fill={st.liquid} />
      <path d={`M${cx - 7} ${cy + 8} L${cx + 2} ${cy - 1}`} stroke="#FFFFFF" strokeWidth={1.4} strokeLinecap="round" />
      <rect x={x + 8} y={y + h - 13} width={w - 16} height={3} rx={1.5} fill={st.deep} opacity={0.35} />
    </g>
  );
}

function Shape({ kind, st, clip }: { kind: CategoryStyle["kind"]; st: CategoryStyle; clip: string }) {
  switch (kind) {
    case "bottle": // botol jamu: leher panjang, tutup gabus
      return (
        <g>
          <clipPath id={clip}>
            <path d="M70 112c0-13 10-18 15-20V76h30v16c5 2 15 7 15 20v102c0 8-6 12-13 12H83c-7 0-13-4-13-12Z" />
          </clipPath>
          <path d="M70 112c0-13 10-18 15-20V76h30v16c5 2 15 7 15 20v102c0 8-6 12-13 12H83c-7 0-13-4-13-12Z" fill={GLASS} opacity={0.4} />
          <rect x="60" y="128" width="80" height="110" fill={st.liquid} clipPath={`url(#${clip})`} />
          <rect x="84" y="60" width="32" height="18" rx="4" fill={st.deep} />
          <Label x={77} y={146} w={46} h={56} st={st} />
          <rect x="76" y="104" width="5" height="100" rx="2.5" fill="#fff" opacity={0.55} />
        </g>
      );
    case "jar": // toples madu
      return (
        <g>
          <clipPath id={clip}>
            <rect x="54" y="98" width="92" height="124" rx="24" />
          </clipPath>
          <rect x="54" y="98" width="92" height="124" rx="24" fill={GLASS} opacity={0.4} />
          <rect x="50" y="116" width="100" height="112" fill={st.liquid} clipPath={`url(#${clip})`} />
          <path d="M54 116h92v10c-8 0-8 14-16 14s-8-8-14-8-8 18-16 18-8-12-14-12-8 10-16 10-8-14-16-14Z" fill={st.liquid} clipPath={`url(#${clip})`} opacity={0.9} />
          <rect x="60" y="78" width="80" height="24" rx="7" fill={st.deep} />
          <Label x={70} y={140} w={60} h={52} st={st} />
          <rect x="62" y="108" width="5" height="96" rx="2.5" fill="#fff" opacity={0.55} />
        </g>
      );
    case "oil": // botol minyak: ramping, tutup hitam kecil
      return (
        <g>
          <clipPath id={clip}>
            <path d="M80 112c0-14 10-16 13-20V70h14v22c3 4 13 6 13 20v104c0 8-6 12-13 12H93c-7 0-13-4-13-12Z" />
          </clipPath>
          <path d="M80 112c0-14 10-16 13-20V70h14v22c3 4 13 6 13 20v104c0 8-6 12-13 12H93c-7 0-13-4-13-12Z" fill={GLASS} opacity={0.4} />
          <rect x="70" y="120" width="60" height="120" fill={st.liquid} clipPath={`url(#${clip})`} />
          <rect x="88" y="50" width="24" height="24" rx="6" fill={st.deep} />
          <Label x={86} y={148} w={28} h={50} st={st} />
          <rect x="85" y="108" width="4" height="96" rx="2" fill="#fff" opacity={0.55} />
        </g>
      );
    case "tub": // wadah kosmetik: pendek, tutup lebar
      return (
        <g>
          <clipPath id={clip}>
            <rect x="48" y="152" width="104" height="70" rx="18" />
          </clipPath>
          <rect x="48" y="152" width="104" height="70" rx="18" fill={GLASS} opacity={0.4} />
          <rect x="44" y="160" width="112" height="70" fill={st.liquid} clipPath={`url(#${clip})`} />
          <rect x="42" y="126" width="116" height="32" rx="11" fill={st.deep} />
          <rect x="52" y="133" width="96" height="4" rx="2" fill="#fff" opacity={0.22} />
          <Label x={72} y={164} w={56} h={42} st={st} />
        </g>
      );
    case "tea": // kotak teh dengan tali kantong
      return (
        <g>
          <rect x="58" y="96" width="84" height="130" rx="8" fill={st.liquid} />
          <path d="M58 96h84l-8-14H66Z" fill={st.deep} opacity={0.85} />
          <path d="M100 82c0-18 14-24 30-22" stroke={st.deep} strokeWidth="2.2" fill="none" strokeLinecap="round" />
          <rect x="126" y="52" width="18" height="14" rx="2" fill="#fff" opacity={0.95} transform="rotate(-8 135 59)" />
          <Label x={70} y={124} w={60} h={72} st={{ ...st, liquid: st.liquid }} />
          <rect x="64" y="106" width="5" height="110" rx="2.5" fill="#fff" opacity={0.28} />
        </g>
      );
    default: // daun
      return (
        <g>
          <path d="M100 232V96" stroke={st.deep} strokeWidth="4" strokeLinecap="round" />
          <path d="M100 150C64 148 50 118 52 92c32 2 50 24 48 58Z" fill={st.liquid} />
          <path d="M100 124c34-2 50-28 48-54-32 2-50 24-48 54Z" fill={st.liquid} opacity={0.82} />
          <path d="M100 186c-30-2-44-24-42-46 28 2 44 20 42 46Z" fill={st.liquid} opacity={0.9} />
          <path d="M100 96c-10-14-8-30 0-42 10 12 10 28 0 42Z" fill={st.liquid} />
        </g>
      );
  }
}

// Wadah saja, tanpa latar. Ukuran mengikuti lebar pembungkusnya.
export function Vessel({ category, className }: { category: string; className?: string }) {
  const st = styleFor(category);
  const clip = useId().replace(/:/g, "");
  return (
    <svg viewBox="0 30 200 215" className={className} role="img" aria-label={category} focusable="false">
      <ellipse cx="100" cy="236" rx="46" ry="6" fill={st.deep} opacity={0.2} />
      <Shape kind={st.kind} st={st} clip={clip} />
    </svg>
  );
}
