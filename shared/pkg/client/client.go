package client

type ClientEntity struct {
	PrimaryKey string  `json:"primaryKey"`
	SortKey    string  `json:"sortKey"`
	Phone      *string `json:"phone"`
	Email      *string `json:"email"`
	ClientName string  `json:"clientName"`
}

func (c *ClientEntity) ToClientDto() ClientDto {
	id := c.SortKey
	return ClientDto{
		CreatedBy:  c.PrimaryKey,
		Id:         &id,
		Phone:      c.Phone,
		Email:      c.Email,
		ClientName: c.ClientName,
	}
}

type ClientDto struct {
	CreatedBy  string  `json:"createdBy"`
	Id         *string `json:"id"`
	Phone      *string `json:"phone"`
	Email      *string `json:"email"`
	ClientName string  `json:"clientName"`
}
