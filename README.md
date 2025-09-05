# goBili

一个用 Go 语言编写的 B站视频下载工具，支持单个视频和专辑下载，默认下载最高清晰度。

## 功能特性

- 🎥 支持单个视频下载
- 📚 支持专辑/合集下载
- 🎯 默认下载最高清晰度
- 🎨 支持多种视频格式 (MP4, FLV)
- 🎵 支持单独下载音频或视频
- ⚡ 多线程下载支持
- 🖥️ 命令行界面
- 📊 下载进度显示
- ⚙️ 配置文件支持
- 🔐 **用户认证系统** - 支持二维码登录和Cookie文件登录
- 📱 **终端二维码显示** - 直接在终端中显示二维码，无需打开浏览器
- 🌐 **真实API集成** - 使用B站官方API获取视频流
- 💾 **Cookie管理** - 自动保存和加载登录状态
- ⏱️ **无超时限制** - 支持大文件长时间下载
- 🎬 **视频音频合并** - 自动使用ffmpeg合并视频和音频流

## 安装

### 从源码编译

#### 使用 Makefile (推荐)

```bash
git clone https://github.com/dengmengmian/goBili.git
cd goBili

# 构建当前平台
make build

# 构建常用平台 (Linux/Windows/macOS)
make build-common

# 构建所有支持平台
make build-all

# 创建发布包
make release

# 查看所有可用命令
make help
```

#### 手动编译

```bash
git clone https://github.com/dengmengmian/goBili.git
cd goBili
go mod tidy
go build -o goBili
```

### 使用 Go 安装

```bash
go install github.com/dengmengmian/goBili@latest
```

## 使用方法

### 基本用法

```bash
# 首先登录（支持多种方式）
goBili login                    # 二维码登录
goBili login -c cookies.txt     # 使用Cookie文件登录
goBili login --browser          # 浏览器登录（自动打开浏览器）

# 登出（清除登录状态）
goBili logout                   # 登出（需要确认）
goBili logout --force           # 强制登出（无需确认）

# 下载单个视频
goBili download "https://www.bilibili.com/video/BV1qt4y1X7TW"

# 下载专辑
goBili download "https://www.bilibili.com/bangumi/play/ss33073"
```

### 高级选项

```bash
# 指定输出目录
goBili download -o ./videos "https://www.bilibili.com/video/BV1qt4y1X7TW"

# 指定视频质量
goBili download -q 720p "https://www.bilibili.com/video/BV1qt4y1X7TW"

# 只下载音频
goBili download -a "https://www.bilibili.com/video/BV1qt4y1X7TW"

# 只下载视频
goBili download -v "https://www.bilibili.com/video/BV1qt4y1X7TW"

# 下载指定分P
goBili download -p 1,2,3 "https://www.bilibili.com/video/BV1At41167aj"

# 下载分P范围
goBili download -p 1-5 "https://www.bilibili.com/video/BV1At41167aj"

# 设置下载线程数
goBili download -t 8 "https://www.bilibili.com/video/BV1qt4y1X7TW"

# 详细输出
goBili download -v "https://www.bilibili.com/video/BV1qt4y1X7TW"
```

### 配置文件

创建配置文件 `~/.goBili.yaml`:

```yaml
output: "./downloads"
threads: 4
verbose: false
quality: "best"
format: "mp4"
```

## 命令行选项

### 全局选项

- `-o, --output`: 输出目录 (默认: ./downloads)
- `-t, --threads`: 下载线程数 (默认: 4)
- `-v, --verbose`: 详细输出
- `--config`: 配置文件路径

### 下载选项

- `-q, --quality`: 视频质量 (best, 1080p, 720p, 480p, 360p)
- `-f, --format`: 输出格式 (mp4, flv)
- `-a, --audio-only`: 只下载音频
- `-v, --video-only`: 只下载视频
- `-p, --pages`: 指定分P (例如: 1,2,3 或 1-5 或 all)

## 支持的URL格式

- 单个视频: `https://www.bilibili.com/video/BV1qt4y1X7TW`
- 专辑: `https://www.bilibili.com/bangumi/play/ss33073`
- 分P视频: `https://www.bilibili.com/video/BV1At41167aj?p=1`

## 项目结构

```text
goBili/
├── cmd/                 # 命令行接口
│   ├── root.go         # 根命令
│   └── download.go     # 下载命令
├── parser/             # B站API解析器
│   └── bilibili.go     # 主要解析逻辑
├── downloader/         # 下载器
│   └── downloader.go   # 下载逻辑
├── main.go             # 程序入口
├── go.mod              # Go模块文件
└── README.md           # 说明文档
```

## 开发

### 依赖

- Go 1.21+
- ffmpeg (用于视频音频合并)
- 以下 Go 模块:
  - `github.com/spf13/cobra` - 命令行框架
  - `github.com/spf13/viper` - 配置管理
  - `github.com/sirupsen/logrus` - 日志

## 构建系统

goBili 提供了完整的 Makefile 构建系统，支持多平台编译和发布。

### 可用命令

| 命令 | 描述 |
|------|------|
| `make build` | 构建当前平台版本 |
| `make build-common` | 构建常用平台 (Linux/Windows/macOS) |
| `make build-all` | 构建所有支持平台 |
| `make clean` | 清理构建文件 |
| `make deps` | 安装依赖 |
| `make test` | 运行测试 |
| `make lint` | 代码检查 |
| `make run` | 构建并运行 |
| `make install` | 安装到系统 |
| `make uninstall` | 从系统卸载 |
| `make release` | 创建发布包 |
| `make version` | 显示版本信息 |
| `make help` | 显示帮助信息 |

### 支持的平台

- **Linux**: AMD64, ARM64
- **Windows**: AMD64, ARM64
- **macOS**: AMD64, ARM64 (Apple Silicon)

### 构建示例

```bash
# 快速构建常用平台
make build-common

# 构建所有平台并创建发布包
make release

# 查看构建的文件
ls -la dist/
```

### 版本信息

构建时会自动注入版本信息：
- 版本号
- 构建时间
- Git 提交哈希

```bash
./goBili version
```

### 安装 ffmpeg

**macOS:**
```bash
brew install ffmpeg
```

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install ffmpeg
```

**Windows:**
下载并安装 [ffmpeg](https://ffmpeg.org/download.html)

### 构建

```bash
go mod tidy
go build -o goBili
```

### 测试

```bash
go test ./...
```

## 认证说明

### 登录方式

goBili 支持三种登录方式：

#### 1. 二维码登录（推荐）
```bash
goBili login
# 使用 B站手机APP 扫描终端中显示的二维码
# 二维码会直接在终端中以ASCII艺术形式显示
```

#### 2. 浏览器登录（便捷）
```bash
goBili login --browser
# 自动打开浏览器到B站登录页面，提供详细的Cookie获取指导
```

#### 3. Cookie文件登录（高级）
```bash
goBili login -c cookies.txt
# 使用保存的Cookie文件直接登录
```

#### 4. 登出（清除登录状态）
```bash
goBili logout                   # 登出（需要确认）
goBili logout --force           # 强制登出（无需确认）
```

### Cookie文件格式

创建一个文本文件，包含从浏览器复制的Cookie信息：

```text
# 格式：name	value	domain	path	expires	size	httpOnly	secure
SESSDATA	your_sessdata_value	.bilibili.com	/	2026/3/4 10:21:44	230 B	✓	✓
bili_jct	your_bili_jct_value	.bilibili.com	/	2026/3/4 10:21:44	40 B	✓
DedeUserID	your_user_id	.bilibili.com	/	2026/3/4 10:21:44	18 B	✓
DedeUserID__ckMd5	your_user_id_md5	.bilibili.com	/	2026/3/4 10:21:44	33 B	✓
sid	your_sid_value	.bilibili.com	/	2026/3/4 10:21:44	11 B	✓
```

### 如何获取Cookie

#### 方法一：使用浏览器登录模式（推荐）
```bash
goBili login --browser
```
此命令会：
1. 自动打开浏览器到B站登录页面
2. 提供详细的Cookie获取步骤指导
3. 告诉您如何从开发者工具中复制Cookie

#### 方法二：手动获取Cookie
1. 在浏览器中登录 B站
2. 按 F12 打开开发者工具
3. 切换到 "Application" 或 "存储" 标签
4. 找到 "Cookies" → "https://www.bilibili.com"
5. 复制需要的Cookie值到文件中

### Cookie 管理

- 登录信息保存在 `~/.goBili/cookies.json` 文件中
- 支持自动加载和保存登录状态
- 如果登录过期，工具会提示重新登录
- 使用 `goBili logout` 可以清除当前登录状态
- 使用 `goBili logout --force` 可以强制清除登录状态（无需确认）

## 注意事项

1. 本工具仅用于个人学习和研究目的
2. 请遵守 B站的服务条款和版权规定
3. 下载的内容仅供个人使用，不得用于商业用途
4. 请尊重内容创作者的权益
5. **需要登录** - 某些高质量视频需要登录后才能下载
6. **API限制** - 请合理使用，避免频繁请求导致IP被封

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！

## 协议文件

- [LICENSE](LICENSE) - MIT 开源协议
- [TERMS.md](TERMS.md) - 使用条款
- [PRIVACY.md](PRIVACY.md) - 隐私政策
- [CONTRIBUTING.md](CONTRIBUTING.md) - 贡献指南

## 致谢

- 参考了 [BBDown](https://github.com/nilaoda/BBDown) 项目的设计思路
- 感谢 B站提供的开放API
