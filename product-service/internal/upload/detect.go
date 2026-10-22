package upload

import "net/http"

func detect(data []byte) string {
	return http.DetectContentType(data)
}
