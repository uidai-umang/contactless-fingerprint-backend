package model

type ErrorResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func NewErrorResponse(message string) ErrorResponse {
	return ErrorResponse{Message: message, Data: nil}
}

func NewErrorResponseWithData(message string, data interface{}) ErrorResponse {
	return ErrorResponse{Message: message, Data: data}
}
