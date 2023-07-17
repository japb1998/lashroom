package utils

import "strconv"

func ExtractNumbers(phone string) *string {
	phoneNumber := make([]rune, 0)
	for _, char := range phone {

		if _, err := strconv.Atoi(string(char)); err == nil {
			phoneNumber = append(phoneNumber, char)
		}

	}
	ptrNumber := string(phoneNumber)
	return &ptrNumber
}
