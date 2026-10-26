import { Brand } from "./Brand";

export function Footer() {
  return (
    <footer className="site-footer">
      <div className="container footer-inner">
        <Brand />
        <p>Proyek portofolio. Dibuat dengan Go, PostgreSQL, Redis, dan Next.js.</p>
      </div>
    </footer>
  );
}
