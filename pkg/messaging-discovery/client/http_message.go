package client

import (
	"errors"
	"fmt"
	"net/http"
)

// Exposed Http messages
var (
	MsgEntrySet              = HTTPMessage{Code: http.StatusOK, Message: "wrote a new entry"}
	MsgEntryUpdated          = HTTPMessage{Code: http.StatusOK, Message: "wrote new entry iteration"}
)

// HTTPMessage represents a message to be returned as an http response
type HTTPMessage struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func (h HTTPMessage) String() string {
	return fmt.Sprintf("status code: %d. message: %s", h.Code, h.Message)
}

// ToError returns an error representing the httpMessage for error comparisons, this is preferred for this type instead
// of implementing error
func (h HTTPMessage) ToError() error {
	return errors.New(h.String())
}
