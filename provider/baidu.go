package provider

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

// BaiduProvider 百度云盘提供商
type BaiduProvider struct {
	accessToken string
	httpClient  *http.Client
	baseURL     string
}

// NewBaiduProvider 创建百度云盘提供商
func NewBaiduProvider(tokens map[string]string) (Provider, error) {
	accessToken := tokens["access_token"]
	if accessToken == "" {
		return nil, fmt.Errorf("缺少 access_token")
	}

	return &BaiduProvider{
		accessToken: accessToken,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		baseURL: "https://pan.baidu.com/rest/2.0/xpan",
	}, nil
}

// Name 返回提供商名称
func (b *BaiduProvider) Name() string {
	return "百度云盘"
}

// UploadFile 上传文件到百度云盘
func (b *BaiduProvider) UploadFile(localPath, remotePath string) error {
	// 获取文件信息
	fileInfo, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("获取文件信息失败: %w", err)
	}

	if fileInfo.IsDir() {
		return b.CreateDir(remotePath)
	}

	log.Printf("[%s] 上传文件: %s -> %s", b.Name(), localPath, remotePath)

	// 计算文件 MD5
	fileMD5, err := b.calculateMD5(localPath)
	if err != nil {
		return fmt.Errorf("计算文件哈希失败: %w", err)
	}

	// 获取父目录路径
	remoteDir := filepath.Dir(remotePath)
	fileName := filepath.Base(remotePath)

	// 确保父目录存在
	parentPath, err := b.getOrCreateDir(remoteDir)
	if err != nil {
		return fmt.Errorf("创建父目录失败: %w", err)
	}

	// 预上传
	uploadURL, err := b.preUpload(fileMD5, fileInfo.Size())
	if err != nil {
		return fmt.Errorf("预上传失败: %w", err)
	}

	// 分片上传
	if uploadURL != "" {
		err = b.uploadFile(uploadURL, localPath)
		if err != nil {
			return fmt.Errorf("上传文件失败: %w", err)
		}
	}

	// 创建文件
	err = b.createFile(parentPath, fileName, fileMD5, fileInfo.Size())
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}

	log.Printf("[%s] 上传完成: %s", b.Name(), remotePath)
	return nil
}

// DeleteFile 删除文件
func (b *BaiduProvider) DeleteFile(remotePath string) error {
	fsID, err := b.getFsIDByPath(remotePath)
	if err != nil {
		return err
	}

	if fsID == "" {
		log.Printf("[%s] 文件不存在，跳过删除: %s", b.Name(), remotePath)
		return nil
	}

	url := fmt.Sprintf("%s/file?method=filemanager&opera=delete", b.baseURL)
	data := map[string]interface{}{
		"filelist": fmt.Sprintf("[\"%s\"]", fsID),
	}

	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", url+"&access_token="+b.accessToken, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("删除文件失败: %s", resp.Status)
	}

	log.Printf("[%s] 删除文件: %s", b.Name(), remotePath)
	return nil
}

// CreateDir 创建目录
func (b *BaiduProvider) CreateDir(remotePath string) error {
	_, err := b.getOrCreateDir(remotePath)
	return err
}

// calculateMD5 计算文件 MD5
func (b *BaiduProvider) calculateMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// preUpload 预上传
func (b *BaiduProvider) preUpload(fileMD5 string, fileSize int64) (string, error) {
	url := fmt.Sprintf("%s/file?method=precreate&access_token=%s", b.baseURL, b.accessToken)

	data := map[string]interface{}{
		"path":       "/"+fileMD5, // 临时路径
		"size":       fileSize,
		"content_md5": fileMD5,
		"is":         0,
	}

	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	result := gjson.ParseBytes(body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("预上传失败: %s", result.String())
	}

	// 检查是否需要上传
	// 如果返回的 return_type 为 1，表示秒传成功
	// 如果返回的 return_type 为 2，表示需要上传
	returnType := result.Get("return_type").Int()

	if returnType == 1 {
		// 秒传成功，无需上传
		return "", nil
	}

	// 获取上传地址
	uploadURL := result.Get("dlink").String()
	return uploadURL, nil
}

// uploadFile 上传文件
func (b *BaiduProvider) uploadFile(uploadURL, localPath string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer file.Close()

	req, err := http.NewRequest("PUT", uploadURL, file)
	if err != nil {
		return err
	}

	// 设置切片上传
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("上传失败: %s", resp.Status)
	}

	return nil
}

// createFile 创建文件
func (b *BaiduProvider) createFile(parentPath, fileName, fileMD5 string, fileSize int64) error {
	url := fmt.Sprintf("%s/file?method=create&access_token=%s", b.baseURL, b.accessToken)

	remotePath := strings.TrimSuffix(parentPath, "/")
	if remotePath == "" {
		remotePath = "/"
	}

	remotePath = remotePath + "/" + fileName

	data := map[string]interface{}{
		"path":       remotePath,
		"size":       fileSize,
		"content_md5": fileMD5,
		"is":         0,
	}

	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	result := gjson.ParseBytes(body)

	if resp.StatusCode != http.StatusOK {
		// 检查是否文件已存在
		if result.Get("errno").Int() == -8 {
			// 文件已存在，视为成功
			return nil
		}
		return fmt.Errorf("创建文件失败: %s", result.String())
	}

	return nil
}

// getFsIDByPath 根据路径获取文件 fs_id
func (b *BaiduProvider) getFsIDByPath(remotePath string) (string, error) {
	remotePath = strings.Trim(remotePath, "/")
	if remotePath == "" {
		return "", nil
	}

	// 获取文件元信息
	url := fmt.Sprintf("%s/file?method=meta&access_token=%s", b.baseURL, b.accessToken)
	url += fmt.Sprintf("&path=/%s", remotePath)

	resp, err := b.httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	result := gjson.ParseBytes(body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("获取文件信息失败: %s", result.String())
	}

	if result.Get("errno").Int() != 0 {
		return "", nil
	}

	return result.Get("list.0.fs_id").String(), nil
}

// getOrCreateDir 获取或创建目录
func (b *BaiduProvider) getOrCreateDir(remotePath string) (string, error) {
	remotePath = strings.Trim(remotePath, "/")
	if remotePath == "" {
		return "/", nil
	}

	// 检查目录是否存在
	fsID, err := b.getFsIDByPath(remotePath)
	if err != nil {
		return "", err
	}

	if fsID != "" {
		return "/" + remotePath, nil
	}

	// 创建目录
	return b.createDirRecursive(remotePath)
}

// createDirRecursive 递归创建目录
func (b *BaiduProvider) createDirRecursive(remotePath string) (string, error) {
	parts := strings.Split(remotePath, "/")
	currentPath := ""

	for _, part := range parts {
		if part == "" {
			continue
		}

		if currentPath == "" {
			currentPath = part
		} else {
			currentPath += "/" + part
		}

		// 检查目录是否存在
		fsID, err := b.getFsIDByPath(currentPath)
		if err != nil {
			return "", err
		}

		if fsID == "" {
			// 创建目录
			err := b.createSingleDir("/" + currentPath)
			if err != nil {
				return "", err
			}
		}
	}

	return "/" + remotePath, nil
}

// createSingleDir 创建单个目录
func (b *BaiduProvider) createSingleDir(remotePath string) error {
	url := fmt.Sprintf("%s/file?method=create&access_token=%s", b.baseURL, b.accessToken)

	data := map[string]interface{}{
		"path": remotePath,
		"size": 0,
		"is":   1, // 1 表示目录
	}

	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	result := gjson.ParseBytes(body)

	if resp.StatusCode != http.StatusOK {
		// 检查是否目录已存在
		if result.Get("errno").Int() == -8 {
			return nil
		}
		return fmt.Errorf("创建目录失败: %s", result.String())
	}

	return nil
}
