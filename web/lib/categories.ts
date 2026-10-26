// Gaya visual per kategori: bentuk wadah dan warna. Kategori yang tidak dikenal
// (admin boleh membuat kategori baru) memakai gaya "daun".
export type VesselKind = "bottle" | "jar" | "oil" | "tub" | "tea" | "leaf";

export type CategoryStyle = {
  kind: VesselKind;
  bg: string; // latar gambar produk
  liquid: string; // isi wadah
  deep: string; // tutup, bayangan, garis
};

const STYLES: Record<string, CategoryStyle> = {
  jamu: { kind: "bottle", bg: "#F6DD8A", liquid: "#E58F0B", deep: "#7A4300" },
  madu: { kind: "jar", bg: "#F0B35E", liquid: "#B3620A", deep: "#5A2B00" },
  minyak: { kind: "oil", bg: "#CFE0A8", liquid: "#7D9C2F", deep: "#3A4F12" },
  kosmetik: { kind: "tub", bg: "#F2CFC6", liquid: "#C9695B", deep: "#6B2920" },
  teh: { kind: "tea", bg: "#BFE0D0", liquid: "#3F8F70", deep: "#1D4A3A" },
};

const FALLBACK: CategoryStyle = { kind: "leaf", bg: "#D5E5CC", liquid: "#4F8A55", deep: "#1F4A2A" };

export function styleFor(category: string): CategoryStyle {
  return STYLES[category.trim().toLowerCase()] ?? FALLBACK;
}

// "jamu" -> "Jamu"
export function labelFor(category: string): string {
  const c = category.trim();
  return c ? c[0].toUpperCase() + c.slice(1) : c;
}

// Urutan rak di hero bila API belum menjawab.
export const DEFAULT_SHELF = ["jamu", "madu", "minyak", "kosmetik", "teh"];
