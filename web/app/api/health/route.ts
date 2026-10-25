// Dipakai healthcheck Docker. Sengaja tidak memanggil service lain: "web hidup" tidak boleh
// bergantung pada auth-service atau product-service (halaman publik tetap harus bisa dibuka).
export const dynamic = "force-dynamic";

export function GET() {
  return Response.json({ status: "ok" });
}
