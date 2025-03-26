# 使用官方的Alpine Linux作为基础镜像，因为它非常小
FROM alpine:latest

# 更新包列表并安装telnet和curl
RUN apk update && \
    apk add --no-cache curl busybox-extras

# 设置默认命令为bash（如果需要的话）
CMD ["sh"]
