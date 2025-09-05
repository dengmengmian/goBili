# 贡献指南

感谢您对 goBili 项目的关注！我们欢迎各种形式的贡献。

## 如何贡献

### 1. 报告问题
- 使用 GitHub Issues 报告 Bug
- 提供详细的复现步骤
- 包含系统信息和错误日志
- 检查是否已有相同问题

### 2. 功能建议
- 通过 GitHub Issues 提出新功能建议
- 详细描述功能需求和预期行为
- 说明为什么需要这个功能
- 考虑实现的复杂度和影响

### 3. 代码贡献
- Fork 本仓库
- 创建功能分支 (`git checkout -b feature/AmazingFeature`)
- 提交更改 (`git commit -m 'Add some AmazingFeature'`)
- 推送到分支 (`git push origin feature/AmazingFeature`)
- 创建 Pull Request

## 开发环境设置

### 1. 环境要求
- Go 1.21+
- Git
- Make (可选，用于构建)

### 2. 本地开发
```bash
# 克隆仓库
git clone https://github.com/dengmengmian/goBili.git
cd goBili

# 安装依赖
go mod tidy

# 运行测试
make test

# 构建项目
make build
```

### 3. 代码规范
- 遵循 Go 官方代码规范
- 使用 `gofmt` 格式化代码
- 运行 `go vet` 检查代码
- 添加必要的注释和文档

## 提交规范

### 1. 提交信息格式
```
<type>(<scope>): <subject>

<body>

<footer>
```

### 2. 类型说明
- `feat`: 新功能
- `fix`: Bug 修复
- `docs`: 文档更新
- `style`: 代码格式调整
- `refactor`: 代码重构
- `test`: 测试相关
- `chore`: 构建过程或辅助工具的变动

### 3. 示例
```
feat(auth): add QR code login support

- Implement QR code generation
- Add QR code display in terminal
- Support QR code status polling

Closes #123
```

## 代码审查

### 1. Pull Request 要求
- 代码必须通过所有测试
- 添加必要的测试用例
- 更新相关文档
- 遵循代码规范

### 2. 审查流程
- 至少需要一位维护者审查
- 审查者会检查代码质量和功能正确性
- 可能需要修改后重新提交

## 测试

### 1. 运行测试
```bash
# 运行所有测试
make test

# 运行特定包的测试
go test ./auth

# 运行测试并显示覆盖率
go test -cover ./...
```

### 2. 测试要求
- 新功能必须包含测试用例
- 测试覆盖率不应低于 80%
- 测试应该独立且可重复

## 文档

### 1. 代码文档
- 公共函数和类型必须有注释
- 注释应该说明功能和参数
- 使用 Go 官方文档格式

### 2. 用户文档
- 更新 README.md 中的使用说明
- 添加新功能的示例
- 保持文档的准确性和时效性

## 发布流程

### 1. 版本管理
- 使用语义化版本号 (SemVer)
- 主版本号：不兼容的 API 修改
- 次版本号：向下兼容的功能性新增
- 修订号：向下兼容的问题修正

### 2. 发布检查
- 所有测试通过
- 文档已更新
- 版本号已更新
- 发布说明已准备

## 行为准则

### 1. 社区准则
- 尊重所有贡献者
- 使用友好和包容的语言
- 接受建设性的批评
- 关注对社区最有利的事情

### 2. 禁止行为
- 使用性暗示的语言或图像
- 发表歧视性言论
- 人身攻击或侮辱
- 公开或私下骚扰

## 许可证

通过贡献代码，您同意您的贡献将在 MIT 许可证下发布。

## 联系方式

- GitHub Issues: https://github.com/dengmengmian/goBili
- Email: my@dengmengmian.com

## 致谢

感谢所有为 goBili 项目做出贡献的开发者！

---

**最后更新：2025年9月5日**
