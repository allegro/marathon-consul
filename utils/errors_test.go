package utils

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrors_shouldMergeErrors(t *testing.T) {
	t.Parallel()
	// given
	errs := []error{errors.New("first"), errors.New("second")}

	// when
	err := MergeErrorsOrNil(errs, "testing")

	// then
	assert.EqualError(t, err, "2 errors occured testing\n1: first\n2: second")
}

func TestErrors_shouldReturnNilForEmptyErrors(t *testing.T) {
	t.Parallel()
	// expect
	assert.NoError(t, MergeErrorsOrNil([]error{}, "testing"))
}
