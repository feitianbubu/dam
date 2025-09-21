package example

type ServiceGroup struct {
	CustomerService
	FileUploadAndDownloadService
	BreakpointContinueService
	AttachmentCategoryService
}

var ServiceGroupApp = new(ServiceGroup)
