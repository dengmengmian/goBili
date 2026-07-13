# goBili Makefile
# 支持多平台构建的B站视频下载工具

# 项目信息
APP_NAME = goBili
VERSION = 1.0.0
BUILD_TIME = $(shell date +%Y-%m-%d_%H:%M:%S)
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 构建目录
BUILD_DIR = build
DIST_DIR = dist

# Go 构建参数
LDFLAGS = -ldflags "-s -w -X github.com/dengmengmian/goBili/cmd.Version=$(VERSION) -X github.com/dengmengmian/goBili/cmd.BuildTime=$(BUILD_TIME) -X github.com/dengmengmian/goBili/cmd.GitCommit=$(GIT_COMMIT)"
GO_FLAGS = -trimpath
CGO_ENABLED = 0

# 默认目标
.PHONY: all
all: clean build

# 清理构建文件
.PHONY: clean
clean:
	@echo "🧹 清理构建文件..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR)
	@echo "✅ 清理完成"

# 安装依赖
.PHONY: deps
deps:
	@echo "📦 安装依赖..."
	@go mod tidy
	@go mod download
	@echo "✅ 依赖安装完成"

# 本地构建 (当前平台)
.PHONY: build
build: deps
	@echo "🔨 构建 $(APP_NAME) (当前平台)..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=$(CGO_ENABLED) go build $(GO_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) .
	@echo "✅ 构建完成: $(BUILD_DIR)/$(APP_NAME)"

# 多平台构建
.PHONY: build-all
build-all: deps
	@echo "🚀 开始多平台构建..."
	@mkdir -p $(DIST_DIR)
	@echo "📦 构建 Linux AMD64..."
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=amd64 go build $(GO_FLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-linux-amd64 .
	@echo "📦 构建 Linux ARM64..."
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=arm64 go build $(GO_FLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-linux-arm64 .
	@echo "📦 构建 Windows AMD64..."
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=amd64 go build $(GO_FLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-windows-amd64.exe .
	@echo "📦 构建 Windows ARM64..."
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=arm64 go build $(GO_FLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-windows-arm64.exe .
	@echo "📦 构建 macOS AMD64..."
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=darwin GOARCH=amd64 go build $(GO_FLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-darwin-amd64 .
	@echo "📦 构建 macOS ARM64 (Apple Silicon)..."
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=darwin GOARCH=arm64 go build $(GO_FLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-darwin-arm64 .
	@echo "✅ 多平台构建完成"
	@echo "📁 构建文件位于: $(DIST_DIR)/"
	@ls -la $(DIST_DIR)/

# 快速构建常用平台
.PHONY: build-common
build-common: deps
	@echo "🎯 构建常用平台..."
	@mkdir -p $(DIST_DIR)
	@echo "📦 构建 Linux AMD64..."
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=amd64 go build $(GO_FLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-linux-amd64 .
	@echo "📦 构建 Windows AMD64..."
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=amd64 go build $(GO_FLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-windows-amd64.exe .
	@echo "📦 构建 macOS AMD64..."
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=darwin GOARCH=amd64 go build $(GO_FLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-darwin-amd64 .
	@echo "📦 构建 macOS ARM64 (Apple Silicon)..."
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=darwin GOARCH=arm64 go build $(GO_FLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-darwin-arm64 .
	@echo "✅ 常用平台构建完成"
	@echo "📁 构建文件位于: $(DIST_DIR)/"
	@ls -la $(DIST_DIR)/

# 测试
.PHONY: test
test:
	@echo "🧪 运行测试..."
	@go test -v ./...

# 代码检查
.PHONY: lint
lint:
	@echo "🔍 代码检查..."
	@go vet ./...
	@go fmt ./...

# 运行
.PHONY: run
run: build
	@echo "🏃 运行 $(APP_NAME)..."
	@./$(BUILD_DIR)/$(APP_NAME) --help

# 安装到系统
.PHONY: install
install: build
	@echo "📥 安装到系统..."
	@sudo cp $(BUILD_DIR)/$(APP_NAME) /usr/local/bin/
	@echo "✅ 安装完成"

# 卸载
.PHONY: uninstall
uninstall:
	@echo "📤 卸载..."
	@sudo rm -f /usr/local/bin/$(APP_NAME)
	@echo "✅ 卸载完成"

# 创建发布包
.PHONY: release
release: build-all
	@echo "📦 创建发布包..."
	@mkdir -p $(DIST_DIR)/release
	@for file in $(DIST_DIR)/$(APP_NAME)-*; do \
		if [ -f "$$file" ]; then \
			filename=$$(basename $$file); \
			platform=$$(echo $$filename | sed 's/$(APP_NAME)-//' | sed 's/\.exe//'); \
			echo "📦 打包 $$platform..."; \
			tar -czf $(DIST_DIR)/release/$(APP_NAME)-$$platform.tar.gz -C $(DIST_DIR) $$filename; \
		fi; \
	done
	@echo "✅ 发布包创建完成"
	@echo "📁 发布包位于: $(DIST_DIR)/release/"
	@ls -la $(DIST_DIR)/release/

# 显示帮助信息
.PHONY: help
help:
	@echo "goBili 构建系统"
	@echo ""
	@echo "可用命令:"
	@echo "  make build        - 构建当前平台版本"
	@echo "  make build-all    - 构建所有支持平台"
	@echo "  make build-common - 构建常用平台 (Linux/Windows/macOS)"
	@echo "  make clean        - 清理构建文件"
	@echo "  make deps         - 安装依赖"
	@echo "  make test         - 运行测试"
	@echo "  make lint         - 代码检查"
	@echo "  make run          - 构建并运行"
	@echo "  make install      - 安装到系统"
	@echo "  make uninstall    - 从系统卸载"
	@echo "  make release      - 创建发布包"
	@echo "  make help         - 显示此帮助信息"
	@echo ""
	@echo "支持平台:"
	@echo "  - Linux AMD64/ARM64"
	@echo "  - Windows AMD64/ARM64"
	@echo "  - macOS AMD64/ARM64 (Apple Silicon)"
	@echo ""
	@echo "示例:"
	@echo "  make build-common  # 快速构建常用平台"
	@echo "  make build-all     # 构建所有平台"
	@echo "  make release       # 创建发布包"

# 显示版本信息
.PHONY: version
version:
	@echo "goBili $(VERSION)"
	@echo "构建时间: $(BUILD_TIME)"
	@echo "Git提交: $(GIT_COMMIT)"
