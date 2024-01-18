package dto

type BooksyUserDto struct {
	FirstName string `json:"first_name,omitempty" validate:"required,min=2"`
	LastName  string `json:"last_name,omitempty" validate:"required,min=2"`
	CellPhone string `json:"cell_phone,omitempty" validate:"omitempty,e164"`
	Email     string `json:"email,omitempty" validate:"omitempty,email"`
	IsUser    bool   `json:"is_user,omitempty" validate:"required"`
}
