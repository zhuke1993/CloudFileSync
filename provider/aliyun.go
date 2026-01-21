package provider

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

// AliYunProvider 阿里云盘提供商
type AliYunProvider struct {
	accessToken string
	DriveID     string
	httpClient  *http.Client
	baseURL     string
}

// AliYunConfig 阿里云盘配置
type AliYunConfig struct {
	AccessToken string `json:"access_token"`
	DriveID     string `json:"drive_id"`
}

// NewAliYunProvider 创建阿里云盘提供商
func NewAliYunProvider(tokens map[string]string) (Provider, error) {
	accessToken := tokens["access_token"]
	if accessToken == "" {
		return nil, fmt.Errorf("缺少 access_token")
	}

	driveID := tokens["drive_id"]

	return &AliYunProvider{
		accessToken: accessToken,
		DriveID:     driveID,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		baseURL: "https://openapi.alipan.com",
	}, nil
}

// Name 返回提供商名称
func (a *AliYunProvider) Name() string {
	return "阿里云盘"
}

// UploadFile 上传文件到阿里云盘
func (a *AliYunProvider) UploadFile(localPath, remotePath string) error {
	// 获取文件信息
	fileInfo, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("获取文件信息失败: %w", err)
	}

	if fileInfo.IsDir() {
		return a.CreateDir(remotePath)
	}

	// 计算文件 SHA1
	fileSHA1, err := a.calculateSHA1(localPath)
	if err != nil {
		return fmt.Errorf("计算文件哈希失败: %w", err)
	}

	log.Printf("[%s] 上传文件: %s -> %s", a.Name(), localPath, remotePath)

	// 检查文件是否已存在
	fileID, exists, err := a.checkFileExists(remotePath, fileSHA1)
	if err != nil {
		return fmt.Errorf("检查文件是否存在失败: %w", err)
	}

	if exists {
		log.Printf("[%s] 文件已存在，跳过上传: %s", a.Name(), remotePath)
		return nil
	}

	// 获取上传地址
	uploadURL, err := a.getUploadURL(fileInfo.Name(), fileSHA1, fileInfo.Size())
	if err != nil {
		return fmt.Errorf("获取上传地址失败: %w", err)
	}

	// 分片上传
	partInfoList, err := a.uploadParts(uploadURL, localPath, fileInfo.Size())
	if err != nil {
		return fmt.Errorf("分片上传失败: %w", err)
	}

	// 完成上传
	err = a.completeUpload(fileID, uploadURL, fileSHA1, fileInfo.Name(), fileInfo.Size(), partInfoList)
	if err != nil {
		return fmt.Errorf("完成上传失败: %w", err)
	}

	log.Printf("[%s] 上传完成: %s", a.Name(), remotePath)
	return nil
}

// DeleteFile 删除文件
func (a *AliYunProvider) DeleteFile(remotePath string) error {
	fileID, err := a.getFileIDByPath(remotePath)
	if err != nil {
		return err
	}

	if fileID == "" {
		log.Printf("[%s] 文件不存在，跳过删除: %s", a.Name(), remotePath)
		return nil
	}

	url := fmt.Sprintf("%s/adrive/v1.0/openFile/delete", a.baseURL)
	data := map[string]string{
		"drive_id": a.DriveID,
		"file_id":  fileID,
	}

	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}

	a.setAuthHeader(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("删除文件失败: %s", resp.Status)
	}

	log.Printf("[%s] 删除文件: %s", a.Name(), remotePath)
	return nil
}

// CreateDir 创建目录
func (a *AliYunProvider) CreateDir(remotePath string) error {
	// 清理路径
	remotePath = strings.Trim(remotePath, "/")
	if remotePath == "" {
		remotePath = "/"
	}

	// 如果是根目录，直接返回
	if remotePath == "/" {
		return nil
	}

	// 检查目录是否存在
	parentDir := filepath.Dir(remotePath)
	dirName := filepath.Base(remotePath)

	parentID, err := a.getOrCreateDir(parentDir)
	if err != nil {
		return err
	}

	// 检查当前目录是否存在
	_, exists, err := a.checkFileExists(remotePath, "")
	if err == nil && exists {
		return nil
	}

	// 创建目录
	url := fmt.Sprintf("%s/adrive/v1.0/openFile/create", a.baseURL)
	data := map[string]interface{}{
		"drive_id":       a.DriveID,
		"parent_file_id": parentID,
		"name":           dirName,
		"type":           "folder",
		"check_name_mode": "refuse",
	}

	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}

	a.setAuthHeader(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("创建目录失败: %s", resp.Status)
	}

	log.Printf("[%s] 创建目录: %s", a.Name(), remotePath)
	return nil
}

// calculateSHA1 计算文件 SHA1
func (a *AliYunProvider) calculateSHA1(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha1.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// getUploadURL 获取上传地址
func (a *AliYunProvider) getUploadURL(fileName, fileSHA1 string, fileSize int64) (string, error) {
	url := fmt.Sprintf("%s/adrive/v1.0/openFile/getUploadUrl", a.baseURL)
	data := map[string]interface{}{
		"drive_id":  a.DriveID,
		"file_name": fileName,
		"part_info_list": []map[string]int{
			{"part_number": 1},
		},
		"size":         fileSize,
		"check_name_mode": "overwrite",
	}

	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return "", err
	}

	a.setAuthHeader(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	result := gjson.ParseBytes(body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("获取上传地址失败: %s", result.String())
	}

	return result.Get("upload_url").String(), nil
}

// uploadParts 上传分片
func (a *AliYunProvider) uploadParts(uploadURL, localPath string, fileSize int64) ([]map[string]string, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	req, err := http.NewRequest("PUT", uploadURL, file)
	if err != nil {
		return nil, err
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("上传失败: %s", resp.Status)
	}

	return []map[string]string{
		{"part_number": "1"},
	}, nil
}

// completeUpload 完成上传
func (a *AliYunProvider) completeUpload(fileID, uploadURL, fileSHA1, fileName string, fileSize int64, partInfoList []map[string]string) error {
	url := fmt.Sprintf("%s/adrive/v1.0/openFile/complete", a.baseURL)
	data := map[string]interface{}{
		"drive_id":       a.DriveID,
		"file_id":        fileID,
		"upload_id":      a.extractUploadID(uploadURL),
		"part_info_list": partInfoList,
	}

	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}

	a.setAuthHeader(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("完成上传失败: %s", resp.Status)
	}

	return nil
}

// checkFileExists 检查文件是否存在
func (a *AliYunProvider) checkFileExists(remotePath, contentHash string) (string, bool, error) {
	fileID, err := a.getFileIDByPath(remotePath)
	if err != nil || fileID == "" {
		return "", false, err
	}

	// 获取文件信息
	url := fmt.Sprintf("%s/adrive/v1.0/openFile/get", a.baseURL)
	data := map[string]string{
		"drive_id": a.DriveID,
		"file_id":  fileID,
	}

	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return "", false, err
	}

	a.setAuthHeader(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	result := gjson.ParseBytes(body)

	if resp.StatusCode != http.StatusOK {
		return "", false, nil
	}

	existingHash := result.Get("content_hash").String()
	if existingHash == contentHash {
		return fileID, true, nil
	}

	return fileID, false, nil
}

// getFileIDByPath 根据路径获取文件ID
func (a *AliYunProvider) getFileIDByPath(remotePath string) (string, error) {
	remotePath = strings.Trim(remotePath, "/")
	if remotePath == "" {
		return "root", nil
	}

	parts := strings.Split(remotePath, "/")
	currentID := "root"

	for i, part := range parts {
		if part == "" {
			continue
		}

		fileID, err := a.findFileInDir(currentID, part)
		if err != nil {
			// 如果是最后一级且文件不存在，返回空
			if i == len(parts)-1 {
				return "", nil
			}
			return "", err
		}

		if fileID == "" {
			return "", nil
		}

		currentID = fileID
	}

	return currentID, nil
}

// findFileInDir 在目录中查找文件
func (a *AliYunProvider) findFileInDir(parentID, fileName string) (string, error) {
	url := fmt.Sprintf("%s/adrive/v1.0/openFile/list", a.baseURL)
	data := map[string]string{
		"drive_id":       a.DriveID,
		"parent_file_id": parentID,
		"limit":          "100",
	}

	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return "", err
	}

	a.setAuthHeader(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	result := gjson.ParseBytes(body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("列举文件失败: %s", result.String())
	}

	items := result.Get("items").Array()
	for _, item := range items {
		if item.Get("name").String() == fileName {
			return item.Get("file_id").String(), nil
		}
	}

	return "", nil
}

// getOrCreateDir 获取或创建目录
func (a *AliYunProvider) getOrCreateDir(remotePath string) (string, error) {
	remotePath = strings.Trim(remotePath, "/")
	if remotePath == "" {
		return "root", nil
	}

	fileID, err := a.getFileIDByPath(remotePath)
	if err != nil {
		return "", err
	}

	if fileID != "" {
		return fileID, nil
	}

	// 创建目录
	return a.createDirRecursive(remotePath)
}

// createDirRecursive 递归创建目录
func (a *AliYunProvider) createDirRecursive(remotePath string) (string, error) {
	parts := strings.Split(remotePath, "/")
	currentPath := ""
	currentID := "root"

	for _, part := range parts {
		if part == "" {
			continue
		}

		currentPath += "/" + part
		fileID, err := a.getFileIDByPath(currentPath)
		if err != nil {
			return "", err
		}

		if fileID == "" {
			// 创建目录
			url := fmt.Sprintf("%s/adrive/v1.0/openFile/create", a.baseURL)
			data := map[string]interface{}{
				"drive_id":       a.DriveID,
				"parent_file_id": currentID,
				"name":           part,
				"type":           "folder",
			}

			jsonData, _ := json.Marshal(data)
			req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
			if err != nil {
				return "", err
			}

			a.setAuthHeader(req)
			req.Header.Set("Content-Type", "application/json")

			resp, err := a.httpClient.Do(req)
			if err != nil {
				return "", err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return "", fmt.Errorf("创建目录失败: %s", resp.Status)
			}

			body, _ := io.ReadAll(resp.Body)
			result := gjson.ParseBytes(body)
			currentID = result.Get("file_id").String()
		} else {
			currentID = fileID
		}
	}

	return currentID, nil
}

// extractUploadID 从上传 URL 中提取 upload_id
func (a *AliYunProvider) extractUploadID(uploadURL string) string {
	// 简单实现，实际需要从 URL 中解析
	// 例如: https://cn-beijing-data.aliyundrive.net/...
	// 实际的 upload_id 可能在 URL 参数中
	return "1"
}

// setAuthHeader 设置认证头
func (a *AliYunProvider) setAuthHeader(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+a.accessToken)
}

// UploadWithProgress 带进度的上传（未实现）
func (a *AliYunProvider) uploadWithMultipart(uploadURL string, file *os.File, fileSize int64, progressCallback ProgressCallback) error {
	// 创建 multipart 表单
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filepath.Base(file.Name()))
	if err != nil {
		return err
	}

	uploaded := int64(0)
	buffer := make([]byte, 32*1024) // 32KB 缓冲区

	for {
		n, err := file.Read(buffer)
		if n > 0 {
			part.Write(buffer[:n])
			uploaded += int64(n)

			if progressCallback != nil {
				progressCallback(UploadProgress{
					Uploaded:   uploaded,
					TotalSize:  fileSize,
					Percentage: float64(uploaded) / float64(fileSize) * 100,
				})
			}
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}
	}

	writer.Close()

	req, err := http.NewRequest("PUT", uploadURL, body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("上传失败: %s", resp.Status)
	}

	return nil
}
