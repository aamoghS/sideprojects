package http

const (
	StatusOK                  = 200
	StatusNotFound            = 404
	StatusInternalServerError = 500
)

func statusText(code int) string {
	switch code {
	case StatusOK:
		return "OK"
	case StatusNotFound:
		return "Not Found"
	case StatusInternalServerError:
		return "Internal Server Error"
	default:
		return "Unknown"
	}
}

func Error(w ResponseWriter, msg string, code int) {
	w.WriteHeader(code)
	_, _ = w.Write([]byte(msg))
}
