# CloudFileSync - 云文件同步工具

一个基于 Go 语言开发的文件同步工具，可以监听本地目录的文件变化，并自动将变化的文件上传到阿里云盘或百度云盘。

## 功能特性

- **实时监听**：监听指定目录下的文件变化（创建、修改、删除、重命名）
- **延迟上传**：采用防抖机制，避免文件频繁变化时的重复上传
- **多云支持**：同时支持阿里云盘和百度云盘
- **增量同步**：文件已存在时自动跳过，节省上传时间
- **递归监听**：自动监听子目录的文件变化
- **Web 管理界面**：提供可视化配置管理界面

## 系统要求

- Go 1.21 或更高版本

## 安装

### 1. 克隆项目

```bash
git clone <repository-url>
cd CloudFileSync
```

### 2. 安装依赖

```bash
go mod download
```

### 3. 编译

```bash
go build -o CloudFileSync main.go
```

## 配置

### 1. 创建配置文件

复制示例配置文件：

```bash
cp config.example.json config.json
```

### 2. 编辑配置文件

```json
{
  "watch_dir": "/path/to/watch",  // 要监听的本地目录
  "delay_time": 5,                 // 延迟上传时间（秒）
  "providers": [
    {
      "type": "aliyun",            // 类型: aliyun 或 baidu
      "name": "阿里云盘",           // 显示名称
      "enable": true,              // 是否启用
      "tokens": {
        "access_token": "your_access_token",
        "drive_id": "your_drive_id"
      },
      "target": "/CloudFileSync"   // 云盘目标目录
    }
  ]
}
```

## 获取 Access Token

### 阿里云盘

1. 访问 [阿里云盘开放平台](https://www.alipan.com/drive/open)
2. 创建应用并获取 `access_token` 和 `drive_id`

### 百度云盘

1. 访问 [百度网盘开放平台](https://pan.baidu.com/union/doc/0ksg0sbig)
2. 创建应用并获取 `access_token`

## 使用方法

### 命令行模式

```bash
# 使用默认配置文件 config.json
./CloudFileSync

# 指定配置文件
./CloudFileSync -config /path/to/config.json
```

### Web 管理界面模式

```bash
# 启动 Web 管理界面（默认端口 8080）
./CloudFileSync -web

# 指定端口
./CloudFileSync -web -port 9000
```

启动后在浏览器中访问 `http://localhost:8080` 即可使用可视化配置管理界面。

#### Web 界面功能

- **服务状态监控**：实时查看服务运行状态
- **基本配置**：可视化设置监听目录和延迟时间
- **云盘管理**：添加、编辑、删除云盘配置
- **配置验证**：验证云盘 Token 是否有效
- **操作日志**：查看所有操作记录

### 程序运行

程序启动后会显示：

```
╔══════════════════════════════════════════════════╗
║           CloudFileSync - 云文件同步工具          ║
║              版本: 1.0.0                         ║
╚══════════════════════════════════════════════════╝

配置文件加载成功: config.json
监听目录: /Users/zhuke/Documents/sync
延迟时间: 5 秒
云盘提供商已加载: 阿里云盘 (目标目录: /CloudFileSync)
开始监听目录: /Users/zhuke/Documents/sync
```

当检测到文件变化时：

```
检测到文件变化: /path/to/file.txt [WRITE]
处理文件事件: /path/to/file.txt [WRITE]
[阿里云盘] 上传文件: /path/to/file.txt -> /CloudFileSync/file.txt
[阿里云盘] 上传完成: /CloudFileSync/file.txt
```

### 退出程序

按 `Ctrl+C` 退出程序。

## 项目结构

```
CloudFileSync/
├── main.go                 # 主程序入口
├── config/
│   └── config.go          # 配置管理
├── watcher/
│   └── watcher.go         # 文件监听
├── provider/
│   ├── provider.go        # 云盘接口
│   ├── aliyun.go          # 阿里云盘实现
│   ├── baidu.go           # 百度云盘实现
│   └── factory.go         # 提供商工厂
├── server/
│   └── server.go          # Web 服务器
├── web/
│   ├── index.html         # Web 界面
│   └── static/
│       ├── style.css      # 样式文件
│       └── app.js         # 前端逻辑
├── config.example.json    # 配置文件示例
├── go.mod                 # Go 模块文件
└── README.md              # 项目文档
```

## 技术栈

- **fsnotify**: 文件系统监听库
- **gjson**: JSON 解析库
- **net/http**: HTTP 服务器和客户端
- **原生 HTML/CSS/JavaScript**: 前端界面（无需框架）

## 注意事项

1. **API 限制**：请注意云盘 API 的调用频率限制
2. **大文件上传**：大文件上传可能需要较长时间，请耐心等待
3. **网络稳定性**：建议在稳定的网络环境下使用
4. **权限问题**：确保程序对监听目录有读取权限

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！
