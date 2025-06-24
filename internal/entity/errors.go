package entity

// Error represents the error structure returned by AWS SNS.
type Error struct {
	Type           string `json:"Type"`
	Code           string `json:"Code"`
	Message        string `json:"Message"`
	HTTPStatusCode int    `json:"HttpStatusCode"`
}

// ErrorResponse follows the AWS SNS error response format.
type ErrorResponse struct {
	Error     Error  `json:"Error"`
	RequestID string `json:"RequestId,omitempty"`
}

// Error implements the error interface for ErrorResponse.
func (e ErrorResponse) Error() string {
	return e.Error.Code + ": " + e.Error.Message
}

// Predefined SNS errors for the ListTopics API.
var (
	AuthorizationError = ErrorResponse{
		Error: Error{
			Type:           "Sender",
			Code:           "AuthorizationError",
			Message:        "You are not authorized to perform this action.",
			HTTPStatusCode: 403,
		},
	}

	InternalError = ErrorResponse{
		Error: Error{
			Type:           "Server",
			Code:           "InternalError",
			Message:        "The request failed due to an internal error.",
			HTTPStatusCode: 500,
		},
	}

	InvalidParameter = ErrorResponse{
		Error: Error{
			Type:           "Sender",
			Code:           "InvalidParameter",
			Message:        "One or more parameters are invalid.",
			HTTPStatusCode: 400,
		},
	}
)
