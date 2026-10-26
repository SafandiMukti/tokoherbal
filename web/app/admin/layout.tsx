export const metadata = { title: { default: "Admin", template: "%s | Admin TokoHerbal" } };

// Menu admin ada di header (Kelola produk, Pesanan masuk); layout ini hanya memberi lebar halaman.
export default function AdminLayout({ children }: { children: React.ReactNode }) {
  return <div className="container page">{children}</div>;
}
