FROM alpine:3.19

# 安装必要包
RUN apk add --no-cache \
    bash \
    nginx \
    tzdata \
    curl \
    && ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

# 统一工作目录
WORKDIR /app

# 前端静态资源
COPY build/dist                  /app/dist
COPY web/.docker-compose/nginx/conf.d/nginx.conf /etc/nginx/http.d/default.conf

# 后端可执行文件及资源
COPY server/server           /app/server
COPY build/resource          /app/resource

# 上传目录提前创建
RUN mkdir -p /app/uploads

# 权限与启动脚本
RUN mkdir -p /run/nginx \
    && chown -R nginx:nginx /var/lib/nginx /var/www /run/nginx \
    && chmod +x /app/server

# 启动脚本：先起 nginx，再运行后端
RUN echo '#!/bin/sh' > /app/start.sh \
    && echo 'nginx' >> /app/start.sh \
    && echo 'exec /app/server -c /app/config.release.yaml' >> /app/start.sh \
    && chmod +x /app/start.sh

EXPOSE 80
ENTRYPOINT ["/app/start.sh"]