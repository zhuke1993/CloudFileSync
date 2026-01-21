package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"CloudFileSync/config"
)

// Server Web 服务器
type Server struct {
	config     *config.Config
	configPath string
	httpServer *http.Server
	mu         sync.RWMutex
	// 服务状态
	isRunning bool
}

// Response API 响应
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewServer 创建 Web 服务器
func NewServer(cfg *config.Config, configPath string, port int) *Server {
	s := &Server{
		config:     cfg,
		configPath: configPath,
		isRunning:  false,
	}

	s.httpServer = &http.Server{
		Addr: fmt.Sprintf(":%d", port),
	}

	// 注册路由
	s.setupRoutes()

	return s
}

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	http.HandleFunc("/", s.handleIndex)
	http.HandleFunc("/api/config", s.handleConfig)
	http.HandleFunc("/api/config/save", s.handleSaveConfig)
	http.HandleFunc("/api/providers", s.handleProviders)
	http.HandleFunc("/api/provider/verify", s.handleVerifyProvider)
	http.HandleFunc("/api/service/status", s.handleServiceStatus)
	http.HandleFunc("/api/service/start", s.handleStartService)
	http.HandleFunc("/api/service/stop", s.handleStopService)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
}

// Start 启动服务器
func (s *Server) Start() error {
	log.Printf("Web 界面启动成功: http://localhost%s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Stop 停止服务器
func (s *Server) Stop() error {
	return s.httpServer.Close()
}

// handleIndex 处理首页
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/index.html")
}

// handleConfig 处理配置获取
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	s.sendSuccess(w, "获取配置成功", s.config)
}

// handleSaveConfig 处理配置保存
func (s *Server) handleSaveConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	var newConfig config.Config
	err := json.NewDecoder(r.Body).Decode(&newConfig)
	if err != nil {
		s.sendError(w, "解析配置失败: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 验证配置
	if newConfig.WatchDir == "" {
		s.sendError(w, "监听目录不能为空", http.StatusBadRequest)
		return
	}

	// 检查目录是否存在
	if _, err := os.Stat(newConfig.WatchDir); os.IsNotExist(err) {
		s.sendError(w, "监听目录不存在", http.StatusBadRequest)
		return
	}

	// 保存到文件
	data, err := json.MarshalIndent(newConfig, "", "  ")
	if err != nil {
		s.sendError(w, "序列化配置失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = os.WriteFile(s.configPath, data, 0644)
	if err != nil {
		s.sendError(w, "保存配置失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	s.mu.Lock()
	s.config = &newConfig
	s.mu.Unlock()

	s.sendSuccess(w, "配置保存成功", nil)
}

// handleProviders 处理提供商列表
func (s *Server) handleProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	s.sendSuccess(w, "获取提供商列表成功", s.config.Providers)
}

// handleVerifyProvider 处理提供商验证
func (s *Server) handleVerifyProvider(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Type   string            `json:"type"`
		Tokens map[string]string `json:"tokens"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		s.sendError(w, "解析请求失败: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 这里可以添加实际的验证逻辑
	// 现在简单检查 token 是否存在
	valid := false
	message := "验证失败"

	switch req.Type {
	case "aliyun":
		if req.Tokens["access_token"] != "" {
			valid = true
			message = "验证成功"
		}
	case "baidu":
		if req.Tokens["access_token"] != "" {
			valid = true
			message = "验证成功"
		}
	default:
		message = "不支持的提供商类型"
	}

	if valid {
		s.sendSuccess(w, message, nil)
	} else {
		s.sendError(w, message, http.StatusBadRequest)
	}
}

// handleServiceStatus 处理服务状态
func (s *Server) handleServiceStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	status := map[string]interface{}{
		"running": s.isRunning,
		"watchDir": s.config.WatchDir,
	}

	s.sendSuccess(w, "获取服务状态成功", status)
}

// handleStartService 处理启动服务
func (s *Server) handleStartService(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		s.sendError(w, "服务已在运行", http.StatusConflict)
		return
	}

	s.isRunning = true
	s.sendSuccess(w, "服务启动成功", nil)
}

// handleStopService 处理停止服务
func (s *Server) handleStopService(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		s.sendError(w, "服务未运行", http.StatusConflict)
		return
	}

	s.isRunning = false
	s.sendSuccess(w, "服务停止成功", nil)
}

// sendSuccess 发送成功响应
func (s *Server) sendSuccess(w http.ResponseWriter, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Code:    0,
		Message: message,
		Data:    data,
	})
}

// sendError 发送错误响应
func (s *Server) sendError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(Response{
		Code:    statusCode,
		Message: message,
	})
}

// GetConfigDir 获取配置文件所在目录
func (s *Server) GetConfigDir() string {
	return filepath.Dir(s.configPath)
}
