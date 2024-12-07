package server

// TaskHistory task mapping: <job_id, server_nonce>
type TaskHistory struct {
	JobID       int
	ServerNonce string
}

type Request struct {
	ID     *int                   `json:"id,omitempty"`
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}

type Response struct {
	ID     *int   `json:"id,omitempty"`
	Result bool   `json:"result"`
	Error  string `json:"error"`
}
