# 获取百度云盘 Access Token 方法

## 推荐方法：使用 BaiduPCS-Go

### 1. 安装 BaiduPCS-Go

```bash
# macOS (Intel)
wget https://github.com/qjfoidnh/BaiduPCS-Go/releases/download/v3.9.0/BaiduPCS-Go-v3.9.0-darwin-amd64.zip
unzip BaiduPCS-Go-v3.9.0-darwin-amd64.zip

# macOS (Apple Silicon M1/M2/M3)
wget https://github.com/qjfoidnh/BaiduPCS-Go/releases/download/v3.9.0/BaiduPCS-Go-v3.9.0-darwin-arm64.zip
unzip BaiduPCS-Go-v3.9.0-darwin-arm64.zip
```

### 2. 运行并登录

```bash
./BaiduPCS-Go
```

进入工具后执行：

```
登录
```

工具会显示一个 URL 和授权码：

```
请在浏览器中打开以下链接完成授权：
https://openapi.baidu.com/oauth/...

请输入授权码：XXXX
```

### 3. 获取 Access Token

登录成功后，在以下位置找到 token：

**macOS/Linux:**
```bash
cat ~/.baidupcs-go/cookies.json | grep "access_token"
```

或者直接查看配置文件：
```bash
cat ~/.baidupcs-go/cookies.json
```

你会看到类似这样的内容：
```json
{
  "access_token": "你的access_token",
  "refresh_token": "你的refresh_token",
  ...
}
```

### 4. 复制 Token 到配置文件

将获取的 `access_token` 复制到 `config.json` 中：

```json
{
  "watch_dir": "/Users/zhuke/Documents/sync",
  "delay_time": 5,
  "providers": [
    {
      "type": "baidu",
      "name": "百度云盘",
      "enable": true,
      "tokens": {
        "access_token": "这里粘贴你的token"
      },
      "target": "/CloudFileSync"
    }
  ]
}
```

---

## 方法二：手动获取（需要开发者账号）

如果你已经有百度开发者账号，可以手动获取：

### 1. 创建应用
访问 [百度开放平台](https://open.baidu.com/) 创建网盘应用

### 2. 获取授权码
在浏览器访问：
```
https://openapi.baidu.com/oauth/2.0/authorize?response_type=code&client_id=你的API_KEY&redirect_uri=oob&scope=netdisk
```

### 3. 换取 Token
```bash
curl "https://openapi.baidu.com/oauth/2.0/token" \
-d "grant_type=authorization_code" \
-d "code=上一步的授权码" \
-d "client_id=你的API_KEY" \
-d "client_secret=你的SECRET_KEY" \
-d "redirect_uri=oob"
```

---

## 方法三：使用浏览器开发者工具（临时方法）

1. 登录百度网盘网页版 https://pan.baidu.com/
2. 打开浏览器开发者工具 (F12)
3. 切换到 Network 标签
4. 在网盘中操作任意文件
5. 在请求列表中找到包含 `access_token` 的请求
6. 复制该 token

**注意**: 此方法获取的 token 可能会过期，不如官方 OAuth 方法稳定。

---

## Token 有效期说明

- **Access Token**: 通常有效期为 30 天
- **Refresh Token**: 可以用来刷新 access_token
- **过期后**: 需要重新获取或使用 refresh_token 刷新

建议定期检查并更新 token。
