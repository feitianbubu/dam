package request

type ExaFileIDRequest struct {
	ID uint `json:"ID" binding:"required"`
}
