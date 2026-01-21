#!/usr/bin/env python3
"""
百度网盘 OAuth 授权工具
使用方法: python3 get_baidu_token.py
"""

import webbrowser
import http.server
import socketserver
import urllib.parse
import requests
from threading import Thread
import time

# 百度网盘 OAuth 配置
CLIENT_ID = "8Q4pf1s8G1iG9l1m1nP2qO3t1G2fS3l1"
CLIENT_SECRET = ""  # 需要替换为实际的 secret key
REDIRECT_URI = "http://localhost:8000/callback"

class CallbackHandler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path.startswith('/callback'):
            # 解析授权码
            query = urllib.parse.urlparse(self.path).query
            params = urllib.parse.parse_qs(query)

            if 'code' in params:
                auth_code = params['code'][0]

                # 发送响应页面
                self.send_response(200)
                self.send_header('Content-type', 'text/html; charset=utf-8')
                self.end_headers()

                # 获取 token
                token_data = get_token(auth_code)

                response_html = f"""
                <html>
                <head><title>授权成功</title></head>
                <body style="font-family: Arial; padding: 50px; text-align: center;">
                    <h1>✅ 授权成功！</h1>
                    <p>您的 Access Token:</p>
                    <textarea style="width: 80%; height: 100px; padding: 10px;">{token_data}</textarea>
                    <p><strong>请复制上面的 token 并保存到配置文件中</strong></p>
                    <p>您可以关闭此页面</p>
                </body>
                </html>
                """
                self.wfile.write(response_html.encode('utf-8'))

                # 5秒后关闭服务器
                time.sleep(5)
                print("\n✅ 授权成功！请查看浏览器中的 token")
                print(f"\n您的 Access Token:\n{token_data}\n")

    def log_message(self, format, *args):
        pass  # 静默日志

def get_token(auth_code):
    """使用授权码获取 access token"""
    token_url = "https://openapi.baidu.com/oauth/2.0/token"

    params = {
        'grant_type': 'authorization_code',
        'code': auth_code,
        'client_id': CLIENT_ID,
        'client_secret': CLIENT_SECRET,
        'redirect_uri': REDIRECT_URI
    }

    try:
        response = requests.post(token_url, data=params)
        data = response.json()

        if 'access_token' in data:
            return data['access_token']
        else:
            return f"错误: {data.get('error_description', '未知错误')}"
    except Exception as e:
        return f"请求失败: {str(e)}"

def main():
    print("=" * 60)
    print("百度网盘 OAuth 授权工具")
    print("=" * 60)

    # 启动本地服务器
    PORT = 8000
    handler = CallbackHandler

    with socketserver.TCPServer(("", PORT), handler) as httpd:
        # 打开授权页面
        auth_url = f"https://openapi.baidu.com/oauth/2.0/authorize?response_type=code&client_id={CLIENT_ID}&redirect_uri={urllib.parse.quote(REDIRECT_URI)}&scope=netdisk"

        print(f"\n1. 正在打开授权页面...")
        webbrowser.open(auth_url)

        print(f"2. 请在浏览器中完成授权")
        print(f"3. 本地服务器运行在 http://localhost:{PORT}")
        print("\n等待授权回调...\n")

        # 等待回调
        httpd.serve_forever()

if __name__ == "__main__":
    main()
