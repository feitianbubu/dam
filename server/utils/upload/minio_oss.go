package upload

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
)

var MinioClient *Minio // 优化性能，但是不支持动态配置

// 确保 Minio 结构体实现了 OSSWithMetadata 接口
var _ OSSWithMetadata = (*Minio)(nil)

type Minio struct {
	Client *minio.Client
	bucket string
}

func GetMinio(endpoint, accessKeyID, secretAccessKey, bucketName string, useSSL bool) (*Minio, error) {
	if MinioClient != nil {
		return MinioClient, nil
	}
	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL, // Set to true if using https
	})
	if err != nil {
		return nil, err
	}
	// 尝试创建bucket
	err = minioClient.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})
	if err != nil {
		// Check to see if we already own this bucket (which happens if you run this twice)
		exists, errBucketExists := minioClient.BucketExists(context.Background(), bucketName)
		if errBucketExists == nil && exists {
			// log.Printf("We already own %s\n", bucketName)
		} else {
			return nil, err
		}
	}
	MinioClient = &Minio{Client: minioClient, bucket: bucketName}
	return MinioClient, nil
}

func (m *Minio) UploadFile(file *multipart.FileHeader) (filePathres, key string, uploadErr error) {
	return m.UploadFileWithMetadata(file, nil)
}

// UploadFileWithMetadata 上传文件并支持自定义元数据
func (m *Minio) UploadFileWithMetadata(file *multipart.FileHeader, metadata map[string]string) (filePathres, key string, uploadErr error) {
	return m.UploadFileWithMetadataAndTags(file, metadata, nil)
}

// UploadFileWithMetadataAndTags 上传文件并支持自定义元数据和标签
func (m *Minio) UploadFileWithMetadataAndTags(file *multipart.FileHeader, metadata map[string]string, tags map[string]string) (filePathres, key string, uploadErr error) {
	f, openError := file.Open()
	// mutipart.File to os.File
	if openError != nil {
		global.GVA_LOG.Error("function file.Open() Failed", zap.Any("err", openError.Error()))
		return "", "", errors.New("function file.Open() Failed, err:" + openError.Error())
	}

	filecontent := bytes.Buffer{}
	_, err := io.Copy(&filecontent, f)
	if err != nil {
		global.GVA_LOG.Error("读取文件失败", zap.Any("err", err.Error()))
		return "", "", errors.New("读取文件失败, err:" + err.Error())
	}
	f.Close() // 创建文件 defer 关闭

	// 对文件名进行加密存储
	ext := filepath.Ext(file.Filename)
	filename := utils.MD5V([]byte(strings.TrimSuffix(file.Filename, ext))) + ext
	if global.GVA_CONFIG.Minio.BasePath == "" {
		filePathres = "uploads" + "/" + time.Now().Format("2006-01-02") + "/" + filename
	} else {
		filePathres = global.GVA_CONFIG.Minio.BasePath + "/" + time.Now().Format("2006-01-02") + "/" + filename
	}
	filePathres = filename

	// 根据文件扩展名检测 MIME 类型
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// 构建用户元数据
	userMetadata := make(map[string]string)
	if metadata != nil {
		// 将所有元数据添加前缀，符合MinIO的元数据命名规范
		for key, value := range metadata {
			// MinIO要求用户元数据必须以"x-amz-meta-"开头
			userMetadata["x-amz-meta-"+key] = value
		}
	}

	// 构建标签字符串（MinIO 使用 URL-encoded 格式）
	var userTags map[string]string
	if tags != nil && len(tags) > 0 {
		userTags = tags
	}

	// 设置超时10分钟
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancel()

	// Upload the file with PutObject   大文件自动切换为分片上传
	putOptions := minio.PutObjectOptions{
		ContentType:  contentType,
		UserMetadata: userMetadata,
		UserTags:     userTags,
	}

	info, err := m.Client.PutObject(ctx, global.GVA_CONFIG.Minio.BucketName, filePathres, &filecontent, file.Size, putOptions)
	if err != nil {
		global.GVA_LOG.Error("上传文件到minio失败", zap.Any("err", err.Error()))
		return "", "", errors.New("上传文件到minio失败, err:" + err.Error())
	}

	global.GVA_LOG.Info("文件上传到MinIO成功",
		zap.String("objectName", info.Key),
		zap.Any("metadata", metadata),
		zap.Any("tags", tags))

	return global.GVA_CONFIG.Minio.BucketUrl + "/" + info.Key, filePathres, nil
}

func (m *Minio) DeleteFile(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// Delete the object from MinIO
	err := m.Client.RemoveObject(ctx, m.bucket, key, minio.RemoveObjectOptions{})
	return err
}
