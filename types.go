package fookie

type GraphQLRequest struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName,omitempty"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
}

type GraphQLResponse struct {
	Data   interface{}    `json:"data"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

type GraphQLError struct {
	Message string `json:"message"`
}

func (e GraphQLError) Error() string { return e.Message }

type SubscriptionEvent struct {
	Data  map[string]interface{}
	Error error
}

type EntityEvent struct {
	Op          string `json:"op"`
	Model       string `json:"model"`
	ID          string `json:"id"`
	PayloadJSON string `json:"payload_json"`
	Ts          string `json:"ts"`
}
