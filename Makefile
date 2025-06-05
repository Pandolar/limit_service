# Makefile for Limit Service Go项目

# 变量定义
APP_NAME=limit_service
VERSION=1.0.0
BUILD_DIR=build
DOCKER_IMAGE=$(APP_NAME):$(VERSION)

# Go相关变量
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# 默认目标
.PHONY: all
all: clean deps build

# 安装依赖
.PHONY: deps
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# 构建项目
.PHONY: build
build:
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(APP_NAME) -v ./main.go

# 构建Linux版本
.PHONY: build-linux
build-linux:
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(APP_NAME)-linux -v ./main.go

# 运行项目
.PHONY: run
run:
	$(GOCMD) run main.go

# 运行测试
.PHONY: test
test:
	$(GOTEST) -v ./tests/...

# 运行性能测试
.PHONY: bench
bench:
	$(GOTEST) -bench=. -benchmem ./tests/...

# 运行测试并生成覆盖率报告
.PHONY: test-coverage
test-coverage:
	$(GOTEST) -coverprofile=coverage.out ./tests/...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# 代码格式化
.PHONY: fmt
fmt:
	$(GOCMD) fmt ./...

# 代码检查
.PHONY: vet
vet:
	$(GOCMD) vet ./...

# 清理构建文件
.PHONY: clean
clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Docker相关命令
.PHONY: docker-build
docker-build:
	docker build -t $(DOCKER_IMAGE) .

.PHONY: docker-run
docker-run:
	docker run -p 19892:19892 --env-file .env $(DOCKER_IMAGE)

.PHONY: docker-clean
docker-clean:
	docker rmi $(DOCKER_IMAGE) || true

# 开发环境设置
.PHONY: dev-setup
dev-setup: deps
	@echo "设置开发环境..."
	@if [ ! -f .env ]; then \
		echo "创建 .env 文件..."; \
		echo "REDIS_HOST=localhost" > .env; \
		echo "REDIS_PORT=6379" >> .env; \
		echo "REDIS_PASSWORD=" >> .env; \
		echo "REDIS_DB=0" >> .env; \
	fi
	@echo "开发环境设置完成"

# 生产构建
.PHONY: prod-build
prod-build: clean deps test build-linux
	@echo "生产环境构建完成"

# 帮助信息
.PHONY: help
help:
	@echo "可用的make命令："
	@echo "  all          - 清理、安装依赖并构建"
	@echo "  deps         - 安装Go依赖"
	@echo "  build        - 构建项目"
	@echo "  build-linux  - 构建Linux版本"
	@echo "  run          - 运行项目"
	@echo "  test         - 运行测试"
	@echo "  bench        - 运行性能测试"
	@echo "  test-coverage- 运行测试并生成覆盖率报告"
	@echo "  fmt          - 格式化代码"
	@echo "  vet          - 代码检查"
	@echo "  clean        - 清理构建文件"
	@echo "  docker-build - 构建Docker镜像"
	@echo "  docker-run   - 运行Docker容器"
	@echo "  docker-clean - 清理Docker镜像"
	@echo "  dev-setup    - 设置开发环境"
	@echo "  prod-build   - 生产环境构建"
	@echo "  help         - 显示此帮助信息" 