# syntax=docker/dockerfile:1

# ---------- 1. Build frontend ----------
FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# ---------- 2. Build binary ----------
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Hasil build Vite harus ada sebelum `go build`, karena //go:embed all:dist
# membekukan isinya ke dalam binary saat kompilasi.
COPY --from=web /src/web/dist ./web/dist
# CGO_ENABLED=0 memungkinkan binary statis. Ini bisa karena modernc.org/sqlite
# adalah SQLite murni Go — tidak ada libsqlite3 yang perlu ditautkan.
RUN CGO_ENABLED=0 GOOS=linux go build \
      -trimpath -ldflags="-s -w" \
      -o /out/server ./cmd/server

# ---------- 3. Runtime ----------
FROM alpine:3.21
# ca-certificates : panggilan HTTPS keluar (parser dokumen, backup)
# tzdata          : tanpa ini Asia/Makassar tidak bisa di-resolve dan aplikasi
#                   gagal keras di startup (disengaja — lihat config.Load).
#                   Ini juga alasan memakai alpine, bukan scratch.
RUN apk add --no-cache ca-certificates tzdata wget

WORKDIR /app
COPY --from=build /out/server /app/server

# Default di dalam container menunjuk ke volume persisten. Nilai-nilai ini WAJIB
# sejalan dengan mount Storages di Coolify — kalau /data tidak ter-mount,
# seluruh database hilang di redeploy berikutnya (README §13.2).
ENV DB_PATH=/data/app.db \
    ATTACHMENT_DIR=/data/attachments \
    APP_TZ=Asia/Makassar \
    ADDR=:8080 \
    GOMEMLIMIT=96MiB

VOLUME ["/data"]
EXPOSE 8080

# Health check menyentuh database sungguhan, bukan sekadar membuktikan proses
# hidup — container tanpa volume ter-mount harus dianggap tidak sehat.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1

# Binary berjalan sebagai PID 1. Itu aman di sini karena main.go memasang
# handler SIGTERM sendiri; Coolify mengirim SIGTERM saat redeploy dan aplikasi
# menutup dengan rapi sehingga checkpoint WAL SQLite tuntas.
ENTRYPOINT ["/app/server"]
