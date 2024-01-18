package dto

type PaginationData interface {
	TemplateDto | ClientDto | Notification
}
type PaginatedResponse[T PaginationData] struct {
	Data  []T   `json:"data"`
	Limit int   `json:"limit"`
	Page  int   `json:"page"`
	Total int64 `json:"total"`
}

type PaginationOps struct {
	Page  int  `form:"page" json:"page" binding:"omitempty,min=0"`
	Limit *int `form:"limit" json:"limit" binding:"omitempty,min=1"`
}
