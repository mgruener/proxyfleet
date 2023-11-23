package ipify

import (
	"io"
	"net/http"
)

func MyIP() (string, error) {
	res, err := http.Get("https://api.ipify.org")
	if err != nil {
		return "", err
	}
	ip, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(ip), nil
}
