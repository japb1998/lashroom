package controller

import (
	"github.com/go-playground/validator"
)

// Error Message for Validation Errors
type ErrMsg struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func getErrorMsg(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "This field is required"
	case "lte":
		return "Should be less than " + fe.Param()
	case "gte":
		return "Should be greater than " + fe.Param()
	case "min":
		return "should have min value of " + fe.Param()
	case "e164":
		return "should meet e164 format"
	case "rfc3339":
		return "field should be date" + fe.Param()
	}

	return "Unknown error"
}
