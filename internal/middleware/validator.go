package middleware

import "github.com/go-playground/validator/v10"

type customValidator struct {
	validator *validator.Validate
}

func NewCustomValidator() *customValidator {
	return &customValidator{validator: validator.New()}
}

func (cv *customValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}
