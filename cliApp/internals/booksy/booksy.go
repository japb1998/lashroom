package booksy

type BooksyClient struct {
	Id        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	CellPhone string `json:"cell_phone"`
	Email     string `json:"email"`
}
