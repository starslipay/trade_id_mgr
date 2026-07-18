# 多阶段构建减小镜像体积
FROM docker.m.daocloud.io/golang:1.25-alpine AS builder
WORKDIR /app
# 新增国内GOPROXY配置
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOPROXY=https://goproxy.cn,direct

COPY go.mod go.sum ./
RUN go mod download
COPY . .
# 编译
RUN go build -o trade_id_mgr .

# 运行镜像
FROM docker.m.daocloud.io/alpine:3.19
WORKDIR /app
COPY --from=builder /app/trade_id_mgr .
COPY --from=builder /app/etc ./etc
# 时区
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && apk add --no-cache tzdata
ENV TZ=Asia/Shanghai
EXPOSE 8888
CMD ["./trade_id_mgr", "-f", "./etc/tradeidmgr.yaml"]
