package provider

import "io"

// Provider 云盘提供商接口
type Provider interface {
	// UploadFile 上传文件
	UploadFile(localPath, remotePath string) error

	// DeleteFile 删除文件
	DeleteFile(remotePath string) error

	// CreateDir 创建目录
	CreateDir(remotePath string) error

	// Name 提供商名称
	Name() string
}

// UploadProgress 上传进度回调
type UploadProgress struct {
	FilePath    string
	TotalSize   int64
	Uploaded    int64
	Percentage  float64
}

// ProgressCallback 进度回调函数类型
type ProgressCallback func(progress UploadProgress)

// FileToUpload 待上传文件信息
type FileToUpload struct {
	LocalPath  string
	RemotePath string
	Size       int64
	Reader     io.Reader
}
