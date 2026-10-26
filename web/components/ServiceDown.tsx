export function ServiceDown({ title, message }: { title: string; message?: string }) {
  return (
    <div className="container page">
      <h1>{title}</h1>
      <p className="notice" role="alert">
        {message ?? "Layanan login sedang tidak tersedia. Sesi Anda aman, coba muat ulang beberapa saat lagi."}
      </p>
    </div>
  );
}
