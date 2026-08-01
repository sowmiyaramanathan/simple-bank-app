package validator

import (
	"fmt"
	"net/mail"
	"regexp"
)

var (
	isValidUsername = regexp.MustCompile(`^[a-z0-9_]+$`).MatchString
	isValidFullname = regexp.MustCompile(`^[a-zA-Z\\s]+$`).MatchString
)

func ValidateString(val string, min, max int) (err error) {
	valLen := len(val)
	if valLen < min || valLen > max {
		return fmt.Errorf("must contain between %d - %d characters", min, max)
	}

	return
}

func ValidateUsername(val string) (err error) {
	if err := ValidateString(val, 3, 100); err != nil {
		return err
	}

	if !isValidUsername(val) {
		return fmt.Errorf("must contain only lowercase letters, digits or underscore")
	}

	return
}

func ValidatePassword(val string) (err error) {
	return ValidateString(val, 6, 100)
}

func ValidateFullName(val string) (err error) {
	if err := ValidateString(val, 3, 100); err != nil {
		return err
	}

	if !isValidFullname(val) {
		return fmt.Errorf("must contain only letters or spaces")
	}

	return
}

func ValidateEmail(val string) (err error) {
	if err := ValidateString(val, 3, 200); err != nil {
		return err
	}

	if _, err := mail.ParseAddress(val); err != nil {
		return fmt.Errorf("must contain only lowercase letters, digits or underscore")
	}

	return
}

func ValidateEmailId(value int64) error {
	if value <= 0 {
		return fmt.Errorf("must be a positive integer")
	}
	return nil
}

func ValidateSecretCode(value string) error {
	return ValidateString(value, 32, 128)
}
