package mapper

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/japb1998/control-tower/internal/dto"
	"github.com/japb1998/control-tower/internal/model"
)

func BooksyUserToItem(creator string, u dto.BooksyUserDto) model.ClientItem {

	// parse phone number
	pn := mustParsePhoneNumber(u.CellPhone)

	return *model.NewClientItem(
		creator,
		pn,
		strings.ToLower(u.Email),
		strings.ToLower(u.FirstName),
		strings.ToLower(u.LastName),
		"Added from booksy",
		nil,
	)
}

func mustParsePhoneNumber(n string) string {
	if n == "" {
		return ""
	}

	reg := regexp.MustCompile(`([0-9])+`)
	matchSlc := reg.FindAll([]byte(n), -1)

	p := make([]byte, 0)

	for _, b := range matchSlc {
		p = append(p, b...)
	}

	return fmt.Sprintf("+1%s", string(p))
}
