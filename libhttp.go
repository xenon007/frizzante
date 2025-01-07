package frizzante

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

var client http.Client

// HttpGet sends an http request using the GET verb.
func HttpGet(path string) (string, error) {
	return HttpGetWithHeader(path, map[string]string{})
}

// HttpGetWithHeader sends an http request using the GET verb.
func HttpGetWithHeader(path string, header map[string]string) (string, error) {
	request, requestError := http.NewRequest("GET", path, nil)
	if requestError != nil {
		return "", requestError
	}

	for key, value := range header {
		request.Header.Set(key, value)
	}

	response, doError := client.Do(request)
	if doError != nil {
		return "", doError
	}
	defer response.Body.Close()

	bodyBytes, readError := io.ReadAll(response.Body)
	if readError != nil {
		return "", readError
	}
	result := string(bodyBytes)

	if response.StatusCode >= 300 {
		return result, fmt.Errorf("server responded with status code '%d'", response.StatusCode)
	}

	return result, nil
}

// HttpDelete sends an http request using the DELETE verb.
func HttpDelete(path string) error {
	return HttpDeleteWithHeader(path, map[string]string{})
}

// HttpDeleteWithHeader sends an http request using the DELETE verb.
func HttpDeleteWithHeader(path string, header map[string]string) error {
	request, requestError := http.NewRequest("DELETE", path, nil)
	if requestError != nil {
		return requestError
	}

	for key, value := range header {
		request.Header.Set(key, value)
	}

	response, doError := client.Do(request)
	if doError != nil {
		return doError
	}
	defer response.Body.Close()

	if 200 != response.StatusCode {
		return fmt.Errorf("server responded with status code '%d'", response.StatusCode)
	}

	return nil
}

// HttpPost sends an http request using the POST verb.
func HttpPost(path string, contentType string, contents string) (string, error) {
	return PostWithHeader(path, map[string]string{"Content-Type": contentType}, contents)
}

// PostWithHeader sends an http request using the POST verb.
func PostWithHeader(path string, header map[string]string, contents string) (string, error) {
	request, requestError := http.NewRequest("POST", path, bytes.NewBuffer([]byte(contents)))
	if requestError != nil {
		return "", requestError
	}

	for key, value := range header {
		request.Header.Set(key, value)
	}

	response, doError := client.Do(request)
	if doError != nil {
		return "", doError
	}
	defer response.Body.Close()

	if 200 != response.StatusCode {
		return "", fmt.Errorf("server responded with status code '%d'", response.StatusCode)
	}

	bodyBytes, readError := io.ReadAll(response.Body)
	if readError != nil {
		return "", readError
	}

	result := string(bodyBytes)

	return result, nil
}

// HttpPut sends an http request using the PUT verb.
func HttpPut(path string, contentType string, contents string) (string, error) {
	return PutWithHeader(path, map[string]string{"Content-Type": contentType}, contents)
}

// PutWithHeader sends an http request using the PUT verb.
func PutWithHeader(path string, header map[string]string, contents string) (string, error) {
	request, requestError := http.NewRequest("PUT", path, bytes.NewBuffer([]byte(contents)))
	if requestError != nil {
		return "", requestError
	}

	for key, value := range header {
		request.Header.Set(key, value)
	}

	response, doError := client.Do(request)
	if doError != nil {
		return "", doError
	}
	defer response.Body.Close()

	if 200 != response.StatusCode {
		return "", fmt.Errorf("server responded with status code '%d'", response.StatusCode)
	}

	bodyBytes, readError := io.ReadAll(response.Body)
	if readError != nil {
		return "", readError
	}

	result := string(bodyBytes)

	return result, nil
}
