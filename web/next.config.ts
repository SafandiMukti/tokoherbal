import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // keluaran mandiri: image Docker hanya perlu folder .next/standalone, jauh lebih kecil
  output: "standalone",
  experimental: {
    // Server Action secara bawaan membatasi body 1 MB. Unggah gambar boleh 2 MB
    // (batas di product-service), jadi batas di sini sedikit di atasnya.
    serverActions: { bodySizeLimit: "3mb" },
  },
};

export default nextConfig;
