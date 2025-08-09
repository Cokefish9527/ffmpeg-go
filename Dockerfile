# 使用官方Golang镜像作为构建阶段的基础镜像
FROM golang:1.23-alpine AS builder

# 安装git，某些依赖可能需要
RUN apk add --no-cache git

# 设置构建参数
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

# 设置工作目录
WORKDIR /app

# 复制go mod和sum文件
COPY go.mod go.sum ./

# 清理模块缓存并重新整理依赖
RUN go clean -modcache && go mod tidy

# 设置GOPROXY并下载依赖
RUN go env -w GOPROXY=https://proxy.golang.org,direct && \
    go mod download && \
    go mod verify

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -a -installsuffix cgo -o ffmpeg-go cmd/main.go

# 使用轻量级Alpine镜像作为运行时基础镜像
FROM alpine:latest

# 安装必要的工具和FFmpeg
RUN apk add --no-cache \
    ca-certificates \
    ffmpeg \
    tzdata

# 创建非root用户
RUN adduser -D -s /bin/sh appuser

# 设置工作目录
WORKDIR /app

# 从构建阶段复制编译好的二进制文件
COPY --from=builder /app/ffmpeg-go ./ffmpeg-go

# 创建必要的目录
RUN mkdir -p data temp

# 更改文件所有者
RUN chown -R appuser:appuser /app

# 切换到非root用户
USER appuser

# 暴露端口
EXPOSE 8082

# 设置环境变量
ENV PORT=8082
ENV MAX_WORKERS=16

# 启动应用
CMD ["./ffmpeg-go"]