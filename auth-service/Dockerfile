# Versi Go image harus SAMA ATAU LEBIH BARU dari baris "go" di go.mod.
# Ubah di sini bila perlu (cek tag yang tersedia di hub.docker.com/_/golang).
ARG GO_VERSION=1.27

# Tahap build: Go lengkap. Tahap akhir: hanya biner, image kecil dan tanpa shell.
FROM golang:${GO_VERSION} AS build
WORKDIR /src

# salin go.mod dan go.sum dulu agar layer unduh dependensi ter-cache
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# nonroot: proses berjalan sebagai user biasa, bukan root
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
