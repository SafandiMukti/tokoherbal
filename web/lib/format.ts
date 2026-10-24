const rupiah = new Intl.NumberFormat("id-ID", {
  style: "currency",
  currency: "IDR",
  maximumFractionDigits: 0,
});

export function formatRupiah(n: number): string {
  return rupiah.format(n);
}

const STATUS_LABEL: Record<string, string> = {
  pending: "Menunggu pembayaran",
  paid: "Sudah dibayar",
  shipped: "Dikirim",
  completed: "Selesai",
  cancelled: "Dibatalkan",
};

export function statusLabel(status: string): string {
  return STATUS_LABEL[status] ?? status;
}

export function formatDate(iso: string): string {
  return new Date(iso).toLocaleString("id-ID", { dateStyle: "medium", timeStyle: "short", timeZone: "Asia/Jakarta" });
}
