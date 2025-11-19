SHELL = /bin/bash

#SCRIPT_DIR         = $(shell pwd)/etc/script
#请选择golang版本
BUILD_IMAGE_SERVER  = golang:1.24
#请选择node版本
BUILD_IMAGE_WEB     = node:20
#项目名称
PROJECT_NAME        = github.com/flipped-aurora/gin-vue-admin/server
#配置文件目录
CONFIG_FILE         = config.yaml
#镜像仓库命名空间
IMAGE_NAME          = gva
#镜像地址
REPOSITORY          = registry.cn-hangzhou.aliyuncs.com/${IMAGE_NAME}
#镜像版本
TAGS_OPT           ?= latest
PLUGIN             ?= email

VERSION?=$(shell git describe --tags --always --dirty)
BUILD_TIME?=$(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

#容器环境前后端共同打包
build: build-web build-server
	docker run --name build-local --rm -v $(shell pwd):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} ${BUILD_IMAGE_SERVER} make build-local

#容器环境打包前端
build-web:
	docker run --name build-web-local --rm -v $(shell pwd):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} ${BUILD_IMAGE_WEB} make build-web-local

#容器环境打包后端
build-server:
	docker run --name build-server-local --rm -v $(shell pwd):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} ${BUILD_IMAGE_SERVER} make build-server-local

#构建web镜像
build-image-web:
	@cd web/ && docker build -t ${REPOSITORY}/web:${TAGS_OPT} .

#构建server镜像
build-image-server:
	@cd server/ && docker build -t ${REPOSITORY}/server:${TAGS_OPT} .

#本地环境打包前后端
build-local:
	if [ -d "build" ];then rm -rf build; else echo "OK!"; fi \
	&& if [ -f "/.dockerenv" ];then echo "OK!"; else  make build-web-local && make build-server-local; fi \
	&& mkdir build && cp -r web/dist build/ && cp server/server build/ && cp -r server/resource build/resource
	if [ -f "config.yaml" ]; then cp config.yaml build/; else echo "config.yaml not exist, skip copying"; fi

#本地环境打包前端
build-web-local:
	@cd web/ && if [ -d "dist" ];then rm -rf dist; fi \
	&& yarn config set registry http://mirrors.cloud.tencent.com/npm/ \
	&& if [ ! -d "node_modules" ]; then yarn install; fi \
	&& yarn build

#本地环境打包后端
build-server-local:
	@cd server/ && if [ -f "server" ];then rm -rf server; fi \
	&& go env -w GO111MODULE=on && go env -w GOPROXY=https://goproxy.cn,direct \
	&& go env -w CGO_ENABLED=0 \
	&& if [ ! -d "vendor" ]; then go mod vendor; fi \
	&& go build -buildvcs=false -ldflags "-X main.Version=${VERSION}" -v

#打包前后端二合一镜像
image: swag build
	docker build -t ${REPOSITORY}/gin-vue-admin:${TAGS_OPT} -f deploy/docker/Dockerfile .
	@echo "🐳 为镜像创建标签..."
	@echo "🏷️  Git标签: ${VERSION}"
	docker tag ${REPOSITORY}/gin-vue-admin:${TAGS_OPT} skynono/dam:latest
	docker tag ${REPOSITORY}/gin-vue-admin:${TAGS_OPT} skynono/dam:${VERSION}

#尝鲜版
images: swag build build-image-web build-image-server
	docker build -t ${REPOSITORY}/all:${TAGS_OPT} -f deploy/docker/Dockerfile .

#swagger 文档生成
doc:
	@cd server && swag init
swag:
	$(eval SWAG := build/dist/swag)
	@if ! command -v swag > /dev/null 2>&1; then echo "Installing swag..." && go install github.com/swaggo/swag/cmd/swag@latest; fi
	@cd server && swag init --parseDependency -t Dam,Oidc -o ../${SWAG} --ot=json
	@sed 's/{{\.Version}}/$(VERSION) (Built: $(BUILD_TIME))/g' ${SWAG}/swagger.json > ${SWAG}/swagger.json.tmp \
	 && mv ${SWAG}/swagger.json.tmp ${SWAG}/swagger.json
	@if command -v swagger2openapi > /dev/null 2>&1; then \
		echo "Converting to OpenAPI 3.0..."; \
		swagger2openapi ${SWAG}/swagger.json -o ${SWAG}/openapi3.json; \
	fi
	@cp ${SWAG}/openapi3.json web/public/swag/openapi3.json

#启动调试环境，支持IDEA远程调试
debug:
	@echo "🐛 启动调试环境..."
	@echo "📱 应用端口: http://localhost:8888"
	@echo "🔍 调试端口: localhost:2345"
	@echo "💡 IDEA调试配置："
	@echo "   - Host: localhost"
	@echo "   - Port: 2345"
	@echo "   - 使用 Go Remote 调试模式"
	@echo ""
	docker-compose -f docker-compose.debug.yaml up --build

#停止调试环境
debug-stop:
	@echo "🛑 停止调试环境..."
	docker-compose -f docker-compose.debug.yaml down

#重启调试环境
debug-restart: debug-stop debug

#查看调试环境日志
debug-logs:
	docker-compose -f docker-compose.debug.yaml logs -f server

#仅启动调试服务器（不包含前端和数据库）
debug-server-only:
	@echo "🐛 仅启动后端调试环境..."
	@echo "📱 应用端口: http://localhost:8888"
	@echo "🔍 调试端口: localhost:2345"
	docker-compose -f docker-compose.debug.yaml up --build server

#插件快捷打包： make plugin PLUGIN="这里是插件文件夹名称,默认为email"
plugin:
	if [ -d ".plugin" ];then rm -rf .plugin ; else echo "OK!"; fi && mkdir -p .plugin/${PLUGIN}/{server/plugin,web/plugin} \
	&& if [ -d "server/plugin/${PLUGIN}" ];then cp -r server/plugin/${PLUGIN} .plugin/${PLUGIN}/server/plugin/ ; else echo "OK!"; fi \
	&& if [ -d "web/src/plugin/${PLUGIN}" ];then cp -r web/src/plugin/${PLUGIN} .plugin/${PLUGIN}/web/plugin/ ; else echo "OK!"; fi \
	&& cd .plugin && zip -r ${PLUGIN}.zip ${PLUGIN} && mv ${PLUGIN}.zip ../ && cd ..

push:
	@echo "📤 推送到Docker Hub..."
	docker push skynono/dam:latest
	docker push skynono/dam:${VERSION}
	@echo "✅ Docker镜像推送完成!"
	@echo "🚀 镜像地址:"
	@echo "   - skynono/dam:latest"
	@echo "   - skynono/dam:${VERSION}"
publish: image push
