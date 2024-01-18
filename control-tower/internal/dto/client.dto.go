package dto

type ClientDto struct {
	Id           string  `json:"id"`
	CreatedBy    string  `json:"createdBy"`
	Phone        string  `json:"phone"`
	Email        string  `json:"email"`
	FirstName    string  `json:"firstName"`
	LastName     string  `json:"lastName"`
	LastSeen     *string `json:"lastSeen"`
	CreatedAt    string  `json:"createdAt"`
	LastUpdateAt string  `json:"lastUpdateAt"`
	Description  string  `json:"description"`
	OptIn        *bool   `json:"optIn"`
}

type CreateClient struct {
	Phone        string  `json:"phone" binding:"omitempty,e164"`
	Email        string  `json:"email" binding:"omitempty,email"`
	FirstName    string  `json:"firstName" binding:"required"`
	LastName     string  `json:"lastName" binding:"required"`
	CreatedAt    string  `json:"createdAt" binding:"omitempty,rfc3339"`
	LastUpdateAt string  `json:"lastUpdateAt" binding:"omitempty,rfc3339"`
	Description  string  `json:"description" binding:"omitempty,min=2,max=255"`
	LastSeen     *string `json:"lastSeen" binding:"required,rfc3339"`
}
type PatchClient struct {
	Phone       string  `json:"phone" form:"phone" binding:"omitempty,e164"`
	Email       string  `json:"email" form:"email" binding:"omitempty,email"`
	FirstName   string  `json:"firstName" form:"firstName" binding:"omitempty,min=1"`
	LastName    string  `json:"lastName" form:"lastName" binding:"omitempty,min=1"`
	Description string  `json:"description" form:"description" binding:"omitempty,min=2,max=255"`
	LastSeen    *string `json:"lastSeen" form:"lastSeen" binding:"omitempty,rfc3339"`
	OptIn       *bool   `json:"optIn" form:"optIn" binding:"omitempty,boolean"`
}

type ClientPaginationDto struct {
	PatchClient
	PaginationOps
}
