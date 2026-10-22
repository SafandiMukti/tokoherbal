# Versi Go image harus SAMA ATAU LEBIH BARU dari baris "go" di go.mod.
ARG GO_VERSION=1.27

# Tahap build: Go lengkap. Tahap akhir: hanya biner, image kecil dan tanpa shell.
FROM golang:${GO_VERSION} AS build
WORKDIR /src

# salin go.mod dan go.sum dulu agar layer unduh dependensi ter-cache
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server
# folder untuk gambar unggahan; dibuat di sini karena image akhir tidak punya shell
RUN mkdir -p /out/uploads

# nonroot: proses berjalan sebagai user biasa (UID 65532), bukan root
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server
# folder dimiliki user nonroot; volume bernama yang dipasang ke sini mewarisi kepemilikannya
COPY --from=build --chown=65532:65532 /out/uploads /data/uploads
ENV UPLOAD_DIR=/data/uploads
EXPOSE 8080
ENTRYPOINT ["/server"]
