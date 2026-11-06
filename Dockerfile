# Build stage for Web Studio and Widget
FROM node:20-alpine AS assets-builder
WORKDIR /app

# Build widget
COPY widget/package*.json ./widget/
RUN cd widget && npm ci
COPY widget/ ./widget/
RUN cd widget && npm run build

# Build studio
COPY web/package*.json ./web/
RUN cd web && npm ci
COPY web/ ./web/
RUN cd web && npm run build

# Build stage for Go binary
FROM golang:1.23-alpine AS binary-builder
WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Copy built frontend assets into embedded locations
COPY --from=assets-builder /app/web/dist ./web/dist
COPY --from=assets-builder /app/widget/dist ./internal/widget/dist

# Build pure Go static binary without CGO
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -extldflags '-static'" -o /notiflux ./cmd/notiflux

# Final minimal runtime image (< 25MB)
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app

COPY --from=binary-builder /notiflux /app/notiflux

EXPOSE 8085
VOLUME ["/data"]

ENV NOTIFLUX_PORT=8085 \
    NOTIFLUX_DB_PATH=/data/notiflux.db

ENTRYPOINT ["/app/notiflux"]
CMD ["serve"]
