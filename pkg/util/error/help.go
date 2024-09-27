package error

func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*NotFound); ok {
		return e.Code == 404
	}
	return false
}
