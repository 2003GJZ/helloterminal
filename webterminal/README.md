# WebTerminal

基于 Web 的远程终端管理工具，支持多机器连接和管理。

## 功能特性

- 基于 Web 浏览器的终端访问
- 支持多机器连接管理
- 实时终端输出
- 命令自动补全
- 会话管理
- 权限控制

## 技术栈

### 后端
- Go
- WebSocket
- SSH
- JWT 认证

### 前端
- Vue.js
- Xterm.js
- Element Plus
- Axios

## 项目结构

```
webterminal/
├── frontend/          # 前端代码
│   ├── src/
│   └── package.json
├── backend/           # 后端代码
│   ├── api/          # REST API
│   ├── ws/           # WebSocket 处理
│   └── ssh/          # SSH 连接管理
└── README.md
```

## 开发环境要求

- Go 1.21+
- Node.js 18+
- npm 9+

## 快速开始

1. 启动后端服务
```bash
cd backend
go mod tidy
go run main.go
```

2. 启动前端开发服务器
```bash
cd frontend
npm install
npm run dev
```

## 部署

1. 构建前端
```bash
cd frontend
npm run build
```

2. 构建后端
```bash
cd backend
go build
```

3. 运行
```bash
./webterminal
```
