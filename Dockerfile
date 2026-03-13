FROM node:22-alpine AS frontend-builder
WORKDIR /src/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.25-alpine AS go-builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/wolfword ./main.go

FROM alpine:3.21
RUN addgroup -S app && adduser -S app -G app && apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=go-builder /out/wolfword /app/wolfword
COPY --from=frontend-builder /src/frontend/dist /app/frontend/dist

ENV PORT=3000
ENV DAY_TIMEOUT_SEC=300
ENV VOTE_TIMEOUT_SEC=60

EXPOSE 3000
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 CMD wget -q -O - http://127.0.0.1:${PORT}/healthz >/dev/null || exit 1

USER app
CMD ["./wolfword"]
