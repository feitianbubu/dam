package upload

import (
	"mime/multipart"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// OSS 对象存储接口
// Author [SliverHorn](https://github.com/SliverHorn)
// Author [ccfish86](https://github.com/ccfish86)
type OSS interface {
	UploadFile(file *multipart.FileHeader) (string, string, error)
	DeleteFile(key string) error
}

// OSSWithMetadata 支持元数据的对象存储接口
type OSSWithMetadata interface {
	UploadFile(file *multipart.FileHeader) (string, string, error)
	UploadFileWithMetadata(file *multipart.FileHeader, metadata map[string]string) (string, string, string, error)
	DeleteFile(key string) error
}

// OSSWithACL 支持ACL的对象存储接口
type OSSWithACL interface {
	OSSWithMetadata
	UploadFileWithACL(file *multipart.FileHeader, metadata map[string]string, publicRead bool) (string, string, string, error)
}

// OSSWithACLUpdate 支持更新ACL的对象存储接口
type OSSWithACLUpdate interface {
	OSS
	UpdateObjectACL(key string, publicRead bool) error
}

// OSSWithReplaceFile 支持文件替换的对象存储接口
type OSSWithReplaceFile interface {
	OSS
	ReplaceFile(key string, file *multipart.FileHeader) (string, error)
}

// OSSWithPresignedURL 支持预签名URL的对象存储接口
type OSSWithPresignedURL interface {
	OSS
	GetPresignedURL(key string, expires time.Duration) (string, error)
}

// NewOss OSS的实例化方法
// Author [SliverHorn](https://github.com/SliverHorn)
// Author [ccfish86](https://github.com/ccfish86)
func NewOss() OSS {
	switch global.GVA_CONFIG.System.OssType {
	case "local":
		return &Local{}
	case "qiniu":
		return &Qiniu{}
	case "tencent-cos":
		return &TencentCOS{}
	case "aliyun-oss":
		return &AliyunOSS{}
	case "huawei-obs":
		return HuaWeiObs
	case "aws-s3":
		return &AwsS3{}
	case "cloudflare-r2":
		return &CloudflareR2{}
	case "minio":
		minioClient, err := GetMinio(global.GVA_CONFIG.Minio.Endpoint, global.GVA_CONFIG.Minio.AccessKeyId, global.GVA_CONFIG.Minio.AccessKeySecret, global.GVA_CONFIG.Minio.BucketName, global.GVA_CONFIG.Minio.UseSSL)
		if err != nil {
			global.GVA_LOG.Warn("你配置了使用minio，但是初始化失败，请检查minio可用性或安全配置: " + err.Error())
			panic("minio初始化失败") // 建议这样做，用户自己配置了minio，如果报错了还要把服务开起来，使用起来也很危险
		}
		return minioClient
	default:
		return &Local{}
	}
}
