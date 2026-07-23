// Package response holds the small, generic JSON response shapes used
// outside of application/dto (health/readiness/status acknowledgements).
package response

type StatusResponse struct {
	Status string `json:"status"`
}

type ReadyResponse struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type NotImplementedResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
