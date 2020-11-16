package fetch

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

type FetchInit struct {
	Method  string
	Headers http.Header
	Body    io.Reader
}

type FetchResponse struct {
	OK         bool
	Status     int
	StatusText string
	Body       io.ReadCloser
	Headers    http.Header
}

func Fetch(resource string, init *FetchInit) FetchResponse {
	var req *http.Request
	var err error
	if init == nil {
		req, err = http.NewRequest(http.MethodGet, resource, nil)
	} else {
		req, err = http.NewRequest(init.Method, resource, init.Body)
		if init.Headers != nil {
			for header, value := range init.Headers {
				req.Header[header] = value
			}
		}
	}
	if err != nil {
		return FetchResponse{
			OK:         false,
			Status:     -1,
			StatusText: err.Error(),
		}
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return FetchResponse{
			OK:         false,
			Status:     -1,
			StatusText: err.Error(),
		}
	}
	return FetchResponse{
		OK:         resp.StatusCode >= 200 && resp.StatusCode < 300,
		Status:     resp.StatusCode,
		StatusText: resp.Status[4:],
		Body:       resp.Body,
		Headers:    resp.Header,
	}
}

var errNoBody = errors.New("fetch: response body was nil")

func (resp *FetchResponse) JSON(data interface{}) error {
	if resp.Body == nil {
		return errNoBody
	}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(data); err != nil {
		resp.Body.Close()
		return fmt.Errorf("fetch: %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		return fmt.Errorf("fetch: %v", err)
	}
	return nil
}

func (resp *FetchResponse) Text() (string, error) {
	if resp.Body == nil {
		return "", errNoBody
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		resp.Body.Close()
		return "", fmt.Errorf("fetch: %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		return "", fmt.Errorf("fetch: %v", err)
	}
	return string(data), nil
}

type Error struct {
	URL        string
	Status     int
	StatusText string
	Text       string
}

func (err *Error) Error() string {
	return fmt.Sprintf("fetch: %d %s %q\n%s", err.Status, err.StatusText, err.URL, err.Text)
}

func IsNotExist(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Status == 404
	}
	return false
}
