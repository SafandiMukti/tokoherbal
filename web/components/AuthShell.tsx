import { Vessel } from "./Vessel";

// Dua panel: kiri hijau dengan rak botol (sapaan), kanan kartu formulir.
export function AuthShell({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="auth">
      <aside className="auth-aside on-dark">
        <h2>Ramuan tradisional, tinggal pilih.</h2>
        <p>Masuk untuk menyimpan keranjang dan melacak pesananmu.</p>
        <div className="auth-shelf" aria-hidden="true">
          {["jamu", "madu", "minyak", "teh"].map((c) => (
            <Vessel key={c} category={c} className="vessel" />
          ))}
        </div>
      </aside>
      <div className="auth-main">
        <div className="auth-card">
          <h1>{title}</h1>
          {children}
        </div>
      </div>
    </div>
  );
}
