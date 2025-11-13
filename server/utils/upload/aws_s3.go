package upload

import (
	"errors"
	"mime"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"go.uber.org/zap"
)

type AwsS3 struct{}

//@author: [WqyJh](https://github.com/WqyJh)
//@object: *AwsS3
//@function: UploadFile
//@description: Upload file to Aws S3 using aws-sdk-go. See https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/s3-example-basic-bucket-operations.html#s3-examples-bucket-ops-upload-file-to-bucket
//@param: file *multipart.FileHeader
//@return: string, string, error

func (*AwsS3) UploadFile(file *multipart.FileHeader) (string, string, error) {
	session := newSession()
	uploader := s3manager.NewUploader(session)

	//fileKey := fmt.Sprintf("%d%s", time.Now().Unix(), file.Filename)
	ext := filepath.Ext(file.Filename)
	fileKey := utils.MD5V([]byte(strings.TrimSuffix(file.Filename, ext))) + ext
	filename := global.GVA_CONFIG.AwsS3.PathPrefix + "/" + fileKey
	f, openError := file.Open()
	if openError != nil {
		global.GVA_LOG.Error("function file.Open() failed", zap.Any("err", openError.Error()))
		return "", "", errors.New("function file.Open() failed, err:" + openError.Error())
	}
	defer f.Close() // 创建文件 defer 关闭

	uploadInput := &s3manager.UploadInput{
		Bucket: aws.String(global.GVA_CONFIG.AwsS3.Bucket),
		Key:    aws.String(filename),
		Body:   f,
	}

	// 根据文件扩展名检测 MIME 类型
	//ext := filepath.Ext(file.Filename)
	contentType := mime.TypeByExtension(ext)
	if contentType != "" {
		uploadInput.ContentType = aws.String(contentType)
	}
	_, err := uploader.Upload(uploadInput)
	if err != nil {
		global.GVA_LOG.Error("function uploader.Upload() failed", zap.Any("err", err.Error()))
		return "", "", err
	}

	filename = strings.TrimPrefix(filename, "/")
	return global.GVA_CONFIG.AwsS3.BaseURL + "/" + filename, fileKey, nil
}

//@author: [WqyJh](https://github.com/WqyJh)
//@object: *AwsS3
//@function: DeleteFile
//@description: Delete file from Aws S3 using aws-sdk-go. See https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/s3-example-basic-bucket-operations.html#s3-examples-bucket-ops-delete-bucket-item
//@param: file *multipart.FileHeader
//@return: string, string, error

func (*AwsS3) DeleteFile(key string) error {
	session := newSession()
	svc := s3.New(session)
	filename := global.GVA_CONFIG.AwsS3.PathPrefix + "/" + key
	bucket := global.GVA_CONFIG.AwsS3.Bucket

	_, err := svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filename),
	})
	if err != nil {
		global.GVA_LOG.Error("function svc.DeleteObject() failed", zap.Any("err", err.Error()))
		return errors.New("function svc.DeleteObject() failed, err:" + err.Error())
	}

	_ = svc.WaitUntilObjectNotExists(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filename),
	})
	return nil
}

//@author: [Assistant]
//@object: *AwsS3
//@function: ReplaceFile
//@description: Replace file in Aws S3 using the same key
//@param: key string, file *multipart.FileHeader
//@return: string, error

func (*AwsS3) ReplaceFile(key string, file *multipart.FileHeader) (string, error) {
	session := newSession()
	uploader := s3manager.NewUploader(session)

	filename := global.GVA_CONFIG.AwsS3.PathPrefix + "/" + key
	f, openError := file.Open()
	if openError != nil {
		global.GVA_LOG.Error("function file.Open() failed", zap.Any("err", openError.Error()))
		return "", errors.New("function file.Open() failed, err:" + openError.Error())
	}
	defer f.Close()

	uploadInput := &s3manager.UploadInput{
		Bucket: aws.String(global.GVA_CONFIG.AwsS3.Bucket),
		Key:    aws.String(filename),
		Body:   f,
	}

	// 根据文件扩展名检测 MIME 类型
	ext := filepath.Ext(file.Filename)
	contentType := mime.TypeByExtension(ext)
	if contentType != "" {
		uploadInput.ContentType = aws.String(contentType)
	}

	_, err := uploader.Upload(uploadInput)
	if err != nil {
		global.GVA_LOG.Error("function uploader.Upload() failed", zap.Any("err", err.Error()))
		return "", err
	}

	global.GVA_LOG.Info("文件替换到S3成功",
		zap.String("objectName", filename),
		zap.String("contentType", contentType))

	filename = strings.TrimPrefix(filename, "/")
	return global.GVA_CONFIG.AwsS3.BaseURL + "/" + filename, nil
}

// newSession Create S3 session
func newSession() *session.Session {
	sess, _ := session.NewSession(&aws.Config{
		Region:           aws.String(global.GVA_CONFIG.AwsS3.Region),
		Endpoint:         aws.String(global.GVA_CONFIG.AwsS3.Endpoint), //minio在这里设置地址,可以兼容
		S3ForcePathStyle: aws.Bool(global.GVA_CONFIG.AwsS3.S3ForcePathStyle),
		DisableSSL:       aws.Bool(global.GVA_CONFIG.AwsS3.DisableSSL),
		Credentials: credentials.NewStaticCredentials(
			global.GVA_CONFIG.AwsS3.SecretID,
			global.GVA_CONFIG.AwsS3.SecretKey,
			"",
		),
	})
	return sess
}

//@author: [Assistant]
//@object: *AwsS3
//@function: GetPresignedURL
//@description: Generate a presigned URL for temporary access to a private S3 object
//@param: key string, expires time.Duration
//@return: string, error

func (*AwsS3) GetPresignedURL(key string, expires time.Duration) (string, error) {
	session := newSession()
	svc := s3.New(session)

	filename := global.GVA_CONFIG.AwsS3.PathPrefix + "/" + key

	req, _ := svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(global.GVA_CONFIG.AwsS3.Bucket),
		Key:    aws.String(filename),
	})

	urlStr, err := req.Presign(expires)
	if err != nil {
		global.GVA_LOG.Error("生成预签名URL失败", zap.Any("err", err.Error()))
		return "", errors.New("生成预签名URL失败, err:" + err.Error())
	}

	global.GVA_LOG.Debug("生成预签名URL成功",
		zap.String("key", key),
		zap.Duration("expires", expires))

	return urlStr, nil
}
