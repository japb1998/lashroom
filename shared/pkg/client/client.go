package client

type ClientEntity struct {
	PrimaryKey string  `json:"primaryKey"`
	SortKey    string  `json:"sortKey"`
	Phone      *string `json:"phone"`
	Email      *string `json:"email"`
}

func (c *ClientEntity) ToClientDto() ClientDto {

	return ClientDto{
		CreatedBy: c.PrimaryKey,
		Id:        &c.SortKey,
		Phone:     c.Phone,
		Email:     c.Email,
	}
}

type ClientDto struct {
	CreatedBy string  `json:"createdBy"`
	Id        *string `json:"id"`
	Phone     *string `json:"phone"`
	Email     *string `json:"email"`
}
