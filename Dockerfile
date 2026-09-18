# BirthDayMemo - multi-stage Docker build.
#
# PREREQUISITES (same as a manual build, see README):
#   1. Frontend deps are installed inside the image, nothing to do there.
#   2. PDF fonts: nothing to do. The backend stage below auto-downloads the
#      5 OFL preset fonts (see README/fonts.go) if they are missing, because
#      the Go build embeds `fonts/*.ttf` via go:embed and fails when no font
#      file matches (note: `*.ttf` is git-ignored, so a fresh clone has none).
#      A font you placed manually in internal/pdfexport/fonts/ still wins:
#      it is copied over the download (same filename) before compiling.
#
# Build & run (non-interactive, configured via compose.yaml environment):
#   docker compose up -d --build
#   docker compose logs birthdaymemo | grep -i password   # first start only, if password was generated

# ---------- Stage 1: build the Vue frontend ----------
FROM node:20-alpine AS frontend
WORKDIR /build/frontend

# Install deps first for better layer caching.
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build
# Output: /build/frontend/dist


# ---------- Stage 2: build the Go backend (embeds frontend/dist) ----------
FROM golang:1-alpine AS backend
WORKDIR /src

# Download Go module deps first for better layer caching.
COPY go.mod go.sum ./
RUN go mod download

# PDF preset fonts (all OFL, same 5 as README/internal/pdfexport/fonts.go).
# `*.ttf` is git-ignored, so a fresh clone has no fonts and the go:embed
# pattern `fonts/*.ttf` would match nothing -> build error. Download each
# missing font here (busybox wget ships with this image, no extra package).
# Files placed manually in internal/pdfexport/fonts/ are copied over these
# downloads in the next step (same filename wins), so custom fonts keep working.
RUN mkdir -p internal/pdfexport/fonts && \
    cd internal/pdfexport/fonts && \
    BASE=https://raw.githubusercontent.com/google/fonts/main/ofl && \
    [ -s NotoSansSC.ttf ]   || wget -q -O NotoSansSC.ttf "$BASE/notosanssc/NotoSansSC%5Bwght%5D.ttf" && \
    [ -s ZCOOLXiaoWei.ttf ] || wget -q -O ZCOOLXiaoWei.ttf "$BASE/zcoolxiaowei/ZCOOLXiaoWei-Regular.ttf" && \
    [ -s ZCOOLKuaiLe.ttf ]  || wget -q -O ZCOOLKuaiLe.ttf "$BASE/zcoolkuaile/ZCOOLKuaiLe-Regular.ttf" && \
    [ -s MaShanZheng.ttf ]  || wget -q -O MaShanZheng.ttf "$BASE/mashanzheng/MaShanZheng-Regular.ttf" && \
    [ -s LongCang.ttf ]     || wget -q -O LongCang.ttf "$BASE/longcang/LongCang-Regular.ttf" && \
    ls -la *.ttf

COPY . ./
# Replace any locally built frontend with the fresh multi-arch-safe build.
COPY --from=frontend /build/frontend/dist ./frontend/dist

# Pure-Go SQLite (modernc), so a static CGO-free binary works on alpine/scratch.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/birthdaymemo .


# ---------- Stage 3: minimal runtime ----------
FROM alpine:3
RUN apk add --no-cache ca-certificates tzdata \
 && addgroup -S app && adduser -S app -G app \
 && mkdir -p /opt/birthdaymemo

# Pristine binary, kept OUTSIDE the persistent mount: /opt/birthdaymemo is a
# bind mount from compose.yaml, so anything baked into that path is shadowed.
# docker-entrypoint.sh copies it into the mounted dir on every start so image
# updates take effect even though user data is persistent.
COPY --from=backend /out/birthdaymemo /usr/local/bin/birthdaymemo
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

# The app stores config.json, birthdaymemo.db, logs/ and languages/
# next to the executable, so the entrypoint runs it from the persistent
# /opt/birthdaymemo mount.
USER app

# Default listen port. The actual port is BIRTHDAYMEMO_LISTEN_PORT
# from compose.yaml; keep EXPOSE in sync when changing the default.
EXPOSE 8080

ENTRYPOINT ["docker-entrypoint.sh"]