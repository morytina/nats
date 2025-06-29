package entity

// Error represents the error structure returned by SCP SNS.
type Error struct {
	Type    string `json:"Type"`
	Code    string `json:"Code"`
	Message string `json:"Message"`
}

// ErrorResponse follows the SCP SNS error response format.
type ErrorResponse struct {
	Error     Error  `json:"Error"`
	HTTPCode  int    `json:"HttpStatusCode"`
	RequestID string `json:"RequestId,omitempty"`
}

// Error implements the error interface for ErrorResponse.
func (e ErrorResponse) String() string {
	return e.Error.Code + ": " + e.Error.Message
}

// Predefined SNS errors for the Topics API.
var (
	AuthorizationError = ErrorResponse{
		HTTPCode: 403,
		Error: Error{
			Type:    "Sender",
			Code:    "AuthorizationError",
			Message: "Indicates that the user has been denied access to the requested resource.",
		},
	}

	InternalError = ErrorResponse{
		HTTPCode: 500,
		Error: Error{
			Type:    "Server",
			Code:    "InternalError",
			Message: "Indicates an internal service error.",
		},
	}

	InvalidParameter = ErrorResponse{
		HTTPCode: 400,
		Error: Error{
			Type:    "Sender",
			Code:    "InvalidParameter",
			Message: "Indicates that a request parameter does not comply with the associated constraints.",
		},
	}

	NotFound = ErrorResponse{
		HTTPCode: 404,
		Error: Error{
			Type:    "Sender",
			Code:    "NotFound",
			Message: "Indicates that the requested resource does not exist.",
		},
	}
)
