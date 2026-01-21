package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"

	"CloudFileSync/config"
	"CloudFileSync/provider"
	"CloudFileSync/server"
	"CloudFileSync/watcher"
)

var (
	configFile = flag.String("config", "config.json", "配置文件路径")
	webMode    = flag.Bool("web", false, "启用 Web 管理界面")
	webPort    = flag.Int("port", 8080, "Web 服务器端口")
	version    = "1.0.0"
)

func main() {
	flag.Parse()

	// 打印欢迎信息
	printBanner()

	// 加载配置
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	log.Printf("配置文件加载成功: %s", *configFile)

	// 根据模式选择运行方式
	if *webMode {
		runWebMode(cfg)
	} else {
		runCLIMode(cfg)
	}
}

// runWebMode 运行 Web 模式
func runWebMode(cfg *config.Config) {
	log.Printf("启动 Web 管理界面模式，端口: %d", *webPort)
	log.Printf("请在浏览器中访问: http://localhost:%d", *webPort)

	// 创建 Web 服务器
	srv := server.NewServer(cfg, *configFile, *webPort)

	// 启动服务器
	if err := srv.Start(); err != nil {
		log.Fatalf("Web 服务器启动失败: %v", err)
	}
}

// runCLIMode 运行命令行模式
func runCLIMode(cfg *config.Config) {
	log.Printf("监听目录: %s", cfg.WatchDir)
	log.Printf("延迟时间: %d 秒", cfg.DelayTime)

	// 初始化云盘提供商
	providers := make([]provider.Provider, 0, len(cfg.Providers))
	for _, p := range cfg.Providers {
		if !p.Enable {
			continue
		}

		pvd, err := provider.NewProvider(p)
		if err != nil {
			log.Printf("初始化云盘提供商失败 [%s]: %v", p.Name, err)
			continue
		}

		providers = append(providers, pvd)
		log.Printf("云盘提供商已加载: %s (目标目录: %s)", pvd.Name(), p.Target)
	}

	if len(providers) == 0 {
		log.Fatal("没有可用的云盘提供商，请检查配置文件")
	}

	// 创建文件监听器
	w, err := watcher.NewWatcher(cfg.WatchDir, cfg.GetDelayDuration())
	if err != nil {
		log.Fatalf("创建文件监听器失败: %v", err)
	}
	defer w.Stop()

	// 启动监听
	w.Start()

	// 处理文件变化
	handleFileChanges(w, cfg, providers)

	// 等待退出信号
	waitForExit()
}

// printBanner 打印欢迎信息
func printBanner() {
	fmt.Println(`
╔══════════════════════════════════════════════════╗
║           CloudFileSync - 云文件同步工具          ║
║              版本: ` + version + `                     ║
╚══════════════════════════════════════════════════╝`)
}

// handleFileChanges 处理文件变化
func handleFileChanges(w *watcher.Watcher, cfg *config.Config, providers []provider.Provider) {
	go func() {
		for event := range w.Events() {
			log.Printf("处理文件事件: %s [%s]", event.Path, event.Op)

			// 上传到所有云盘
			var wg sync.WaitGroup
			for i, p := range providers {
				wg.Add(1)
				go func(index int, pvd provider.Provider) {
					defer wg.Done()

					targetDir := cfg.Providers[index].Target
					remotePath := getRemotePath(event.Path, cfg.WatchDir, targetDir)

					err := uploadFile(pvd, event.Path, remotePath, event.Op)
					if err != nil {
						log.Printf("[%s] 上传失败: %v", pvd.Name(), err)
					}
				}(i, p)
			}
			wg.Wait()
		}
	}()
}

// uploadFile 上传文件到云盘
func uploadFile(pvd provider.Provider, localPath, remotePath string, op interface{}) error {
	// 处理删除事件
	// 注意：这里需要根据实际的 fsnotify.Op 类型判断
	opStr := fmt.Sprintf("%v", op)
	if strings.Contains(opStr, "Remove") {
		return pvd.DeleteFile(remotePath)
	}

	// 上传或创建目录
	return pvd.UploadFile(localPath, remotePath)
}

// getRemotePath 获取远程路径
func getRemotePath(localPath, watchDir, targetDir string) string {
	// 获取相对路径
	relPath, err := filepath.Rel(watchDir, localPath)
	if err != nil {
		return filepath.Join(targetDir, filepath.Base(localPath))
	}

	// 确保目标目录以 / 结尾
	targetDir = strings.TrimSuffix(targetDir, "/")
	if targetDir != "" && !strings.HasSuffix(targetDir, "/") {
		targetDir += "/"
	}

	return targetDir + relPath
}

// waitForExit 等待退出信号
func waitForExit() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	<-sigChan
	log.Println("\n收到退出信号，正在关闭...")
}
