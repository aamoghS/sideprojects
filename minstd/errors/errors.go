package errors

import stderrors "errors"

func New(text string) error {
	return stderrors.New(text)
}

func Is(err, target error) bool {
	return stderrors.Is(err, target)
}
