package example

type ServiceGroup struct {
	CustomerService
	FileUploadAndDownloadService
	BreakpointContinueService
	AttachmentCategoryService
	ProjectService
}

var ServiceGroupApp = new(ServiceGroup)
