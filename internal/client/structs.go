package client

type Request struct {
	ID     *int                   `json:"id,omitempty"`
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}
type Task struct {
	JobID       int    `json:"job_id"`
	ServerNonce string `json:"server_nonce"`
}

type Response struct {
	ID     *int   `json:"id"`
	Result bool   `json:"result"`
	Error  string `json:"error"`
}
