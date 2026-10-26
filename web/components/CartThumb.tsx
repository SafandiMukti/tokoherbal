import { imageSrc } from "@/lib/api";
import { ProductArt } from "./ProductArt";

export function CartThumb({ imageUrl, name }: { imageUrl: string; name: string }) {
  const src = imageSrc(imageUrl);
  return (
    <div className="thumb">
      {src ? (
        // eslint-disable-next-line @next/next/no-img-element
        <img className="photo" src={src} alt="" />
      ) : (
        <ProductArt category="" name={name} />
      )}
    </div>
  );
}
