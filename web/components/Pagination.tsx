import Link from "next/link";
import { Icon } from "./Icon";

export function Pagination({
  page,
  limit,
  total,
  params,
  path = "/",
}: {
  page: number;
  limit: number;
  total: number;
  params: Record<string, string>;
  path?: string;
}) {
  const pages = Math.max(1, Math.ceil(total / limit));
  if (pages <= 1) return null;

  const href = (p: number) => {
    const sp = new URLSearchParams(params);
    if (p > 1) sp.set("page", String(p));
    else sp.delete("page");
    const qs = sp.toString();
    return qs ? `${path}?${qs}` : path;
  };

  return (
    <nav className="pagination" aria-label="Halaman">
      {page > 1 ? (
        <Link href={href(page - 1)}>
          <Icon name="chevron-left" size={18} />
          Sebelumnya
        </Link>
      ) : (
        <span />
      )}
      <span className="muted">
        Halaman {page} dari {pages}
      </span>
      {page < pages ? (
        <Link href={href(page + 1)}>
          Berikutnya
          <Icon name="chevron-right" size={18} />
        </Link>
      ) : (
        <span />
      )}
    </nav>
  );
}
