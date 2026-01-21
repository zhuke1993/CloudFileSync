package watcher

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileEvent 文件事件
type FileEvent struct {
	Path      string
	Op        fsnotify.Op
	Timestamp time.Time
}

// Watcher 文件监听器
type Watcher struct {
	fsWatcher *fsnotify.Watcher
	watchDir  string
	eventChan chan FileEvent
	delay     time.Duration
	timerMap  sync.Map // map[string]*time.Timer
	stopChan  chan struct{}
}

// NewWatcher 创建新的文件监听器
func NewWatcher(watchDir string, delay time.Duration) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		fsWatcher: fsWatcher,
		watchDir:  watchDir,
		eventChan: make(chan FileEvent, 100),
		delay:     delay,
		stopChan:  make(chan struct{}),
	}

	// 递归添加监听目录
	err = w.addWatchDir(watchDir)
	if err != nil {
		fsWatcher.Close()
		return nil, err
	}

	return w, nil
}

// addWatchDir 递归添加目录监听
func (w *Watcher) addWatchDir(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// 排除隐藏目录
			if filepath.Base(path)[0] == '.' {
				return filepath.SkipDir
			}
			return w.fsWatcher.Add(path)
		}
		return nil
	})
}

// Start 开始监听
func (w *Watcher) Start() {
	log.Printf("开始监听目录: %s", w.watchDir)

	go func() {
		for {
			select {
			case event, ok := <-w.fsWatcher.Events:
				if !ok {
					return
				}
				w.handleEvent(event)

			case err, ok := <-w.fsWatcher.Errors:
				if !ok {
					return
				}
				log.Printf("监听错误: %v", err)

			case <-w.stopChan:
				return
			}
		}
	}()
}

// handleEvent 处理文件事件
func (w *Watcher) handleEvent(event fsnotify.Event) {
	// 排除临时文件和隐藏文件
	base := filepath.Base(event.Name)
	if len(base) > 0 && base[0] == '.' {
		return
	}

	// 只处理创建、写入、删除、重命名事件
	if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) == 0 {
		return
	}

	log.Printf("检测到文件变化: %s [%s]", event.Name, event.Op)

	// 如果是创建目录，则监听新目录
	if event.Op&fsnotify.Create == fsnotify.Create {
		if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
			w.fsWatcher.Add(event.Name)
			log.Printf("添加新目录监听: %s", event.Name)
		}
	}

	// 重置该文件的定时器（防抖）
	fileEvent := FileEvent{
		Path:      event.Name,
		Op:        event.Op,
		Timestamp: time.Now(),
	}

	// 取消之前的定时器
	if timer, exists := w.timerMap.Load(event.Name); exists {
		timer.(*time.Timer).Stop()
	}

	// 创建新的延迟定时器
	timer := time.AfterFunc(w.delay, func() {
		w.eventChan <- fileEvent
		w.timerMap.Delete(event.Name)
	})

	w.timerMap.Store(event.Name, timer)
}

// Events 返回事件通道
func (w *Watcher) Events() <-chan FileEvent {
	return w.eventChan
}

// Stop 停止监听
func (w *Watcher) Stop() {
	close(w.stopChan)
	w.fsWatcher.Close()

	// 停止所有定时器
	w.timerMap.Range(func(key, value interface{}) bool {
		if timer, ok := value.(*time.Timer); ok {
			timer.Stop()
		}
		w.timerMap.Delete(key)
		return true
	})
}
