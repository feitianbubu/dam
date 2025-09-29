package file_understanding

import (
	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
)

type FileUnderstandingService interface {
	AnalyzeFile(filePath string, fileType string) (*example.FileMetadata, error)

	GetSupportedFileTypes() []string

	IsFileTypeSupported(fileType string) bool
}

type FileProcessor interface {
	ProcessFile(fileID uint) error

	RetryProcessFile(fileID uint) error

	UpdateProcessStatus(fileID uint, status string, metadata *example.FileMetadata, errorMsg string) error
}
