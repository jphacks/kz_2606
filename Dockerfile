# フロントを Go バイナリに埋め込む1デプロイ構成（TECH_PLAN.md 7.5）
FROM node:22-alpine AS web
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
# vite.config.ts の outDir は ../backend/cmd/server/dist
RUN npm run build

FROM golang:1.23-alpine AS build
WORKDIR /src
# go.sum が必要。先に `make tidy` を実行してコミットしておくこと
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=web /app/backend/cmd/server/dist ./cmd/server/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/gpsart ./cmd/server

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/gpsart /gpsart
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/gpsart"]
