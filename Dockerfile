# syntax=docker/dockerfile:1
# ---------- 构建阶段 ----------
FROM golang:1.23-alpine AS build
WORKDIR /src

RUN apk add --no-cache ca-certificates

# 先复制依赖清单（本项目无第三方依赖），后续只复制源码便于分层缓存。
COPY go.mod ./
COPY . .

ARG CGO_ENABLED=0
ENV CGO_ENABLED=${CGO_ENABLED} GOOS=linux GOARCH=amd64

RUN go build -trimpath -ldflags="-s -w" -o /out/elevator-iot-diagnosis-service .

# ---------- 运行阶段 ----------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app \
    && adduser -S app -G app

WORKDIR /app
COPY --from=build /out/elevator-iot-diagnosis-service .

# 数据目录先创建并授权，随后切换非 root 用户。
RUN mkdir -p /app/data && chown -R app:app /app
USER app

# 服务默认监听 8080；容器运行时可用 -e PORT=xxxx 覆盖。
ENV PORT=8080
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q -O /dev/null "http://127.0.0.1:${PORT}/healthz" || exit 1

ENTRYPOINT ["/app/elevator-iot-diagnosis-service"]
