# 多阶段构建 → scratch 单二进制镜像（~3-5MB）
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY engine/ ./engine/
COPY data/ ./data/
COPY backtest/ ./backtest/
COPY fetch/ ./fetch/
COPY report/ ./report/
COPY main.go ./
# 纯标准库，无第三方依赖，CGO 关闭保证静态链接
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /fc3d-kill6 .

FROM scratch
COPY --from=build /fc3d-kill6 /fc3d-kill6
WORKDIR /data
ENTRYPOINT ["/fc3d-kill6"]
