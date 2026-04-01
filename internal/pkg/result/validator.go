package result

import (
	"regexp"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// RegisterCustomValidators registers custom validators matching Java's @CheckForm.
func RegisterCustomValidators() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = v.RegisterValidation("checkform", checkFormValidator)
	}
}

func checkFormValidator(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true // empty is handled by "required" tag
	}
	pattern := fl.Param()
	matched, err := regexp.MatchString(pattern, value)
	if err != nil {
		return false
	}
	return matched
}
