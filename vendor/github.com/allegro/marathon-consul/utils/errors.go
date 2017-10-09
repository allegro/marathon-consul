package utils

import "fmt"

func MergeErrorsOrNil(errors []error, description string) error {
	if len(errors) == 0 {
		return nil
	}

	errMessage := fmt.Sprintf("%d errors occured %s", len(errors), description)
	for i, err := range errors {
		errMessage = fmt.Sprintf("%s\n%d: %s", errMessage, i+1, err.Error())
	}
	return fmt.Errorf(errMessage)
}
