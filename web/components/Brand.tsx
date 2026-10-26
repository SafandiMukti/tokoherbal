import Link from "next/link";
import { Icon } from "./Icon";

export function Brand() {
  return (
    <Link href="/" className="brand" aria-label="TokoHerbal, ke beranda">
      <span className="brand-mark">
        <Icon name="leaf" size={20} />
      </span>
      TokoHerbal
    </Link>
  );
}
