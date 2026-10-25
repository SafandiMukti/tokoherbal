# Versi Node harus sama atau lebih baru dari yang dipakai saat pengembangan (cek: node -v).
ARG NODE_VERSION=24

# 1) dependensi: layer ini ter-cache selama package*.json tidak berubah
FROM node:${NODE_VERSION}-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

# 2) build
FROM node:${NODE_VERSION}-alpine AS build
WORKDIR /app
ENV NEXT_TELEMETRY_DISABLED=1
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

# 3) image akhir: hanya keluaran "standalone" (server kecil + dependensi yang benar-benar dipakai)
FROM node:${NODE_VERSION}-alpine AS run
WORKDIR /app
ENV NODE_ENV=production \
    NEXT_TELEMETRY_DISABLED=1 \
    PORT=3000 \
    HOSTNAME=0.0.0.0
COPY --from=build --chown=node:node /app/.next/standalone ./
COPY --from=build --chown=node:node /app/.next/static ./.next/static
# user "node" sudah ada di image resmi: proses berjalan tanpa hak root
USER node
EXPOSE 3000
CMD ["node", "server.js"]
