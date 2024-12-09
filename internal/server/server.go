package server

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	"luxor.tech/tcp_msg_processing_test/pkg/logger"
	"luxor.tech/tcp_msg_processing_test/pkg/util"
)

type Server struct {
	sessions map[net.Conn]*Session // maintain client sessions
	mu       sync.RWMutex
}

func NewServer() *Server {
	return &Server{
		sessions: make(map[net.Conn]*Session),
	}
}

func (s *Server) Start(port string) error {
	listener, err := net.Listen("tcp", port)
	if err != nil {
		logger.Error("Failed to start server:%v", err)
		return err
	}
	defer listener.Close()

	logger.Info("Server is listening on port:%v", port)

	go s.StartTaskDistribution(time.Second*30, 0)

	// handle client requests: reactor model
	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Info("Error accepting connection:%v", err)
			continue
		}

		logger.Info("New client connected:%v", conn.RemoteAddr())
		s.mu.Lock()
		s.sessions[conn] = NewSession()
		s.mu.Unlock()

		// handle connection
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer func() {
		s.mu.Lock()
		delete(s.sessions, conn)
		s.mu.Unlock()
		conn.Close()
		logger.Info("Client disconnected:%v", conn.RemoteAddr())
	}()

	reader := bufio.NewReader(conn)
	for {
		// read a complete message
		message, err := reader.ReadString('\n') // delimiter based protocol
		if err != nil {
			return
		}
		message = strings.TrimSpace(message)

		// handle one request
		s.processRequest(conn, message)
	}
}

func (s *Server) processRequest(conn net.Conn, message string) {
	var req Request

	err := json.Unmarshal([]byte(message), &req)
	if err != nil {
		logger.Error("Invalid request:%v", err)
		SendErrorResponse(conn, req.ID, "unknown request")
		return
	}

	switch req.Method {
	case "authorize":
		s.mu.RLock()
		session := s.sessions[conn]
		s.mu.RUnlock()
		uname, exist := req.Params["username"]
		if !exist {
			s.mu.Unlock()
			return
		}
		session.Username = uname.(string)
		SendSuccessResponse(conn, req.ID)
	case "submit":
		s.handleSubmit(conn, req)
	}
}

func (s *Server) handleSubmit(conn net.Conn, req Request) {
	// Parse request parameters
	params := req.Params

	jobID, ok := util.IntValue(params["job_id"])
	clientNonce, ok1 := util.StringValue(params["client_nonce"])
	result, ok2 := util.StringValue(params["result"])
	if !ok || !ok1 || !ok2 {
		SendErrorResponse(conn, req.ID, "Missing required parameters")
		return
	}

	// Lock server tasks for thread-safe access
	s.mu.RLock()
	session, e := s.sessions[conn]
	s.mu.RUnlock()
	if !e {
		SendErrorResponse(conn, req.ID, "Task does not exist")
		return
	}
	session.mu.Lock()
	defer session.mu.Unlock()

	// authorize
	if session.Username == "" {
		SendErrorResponse(conn, req.ID, "Not authorized")
		return
	}

	// job_id
	if session.CurrJobID != jobID {
		SendErrorResponse(conn, req.ID, "Task does not exist")
		return
	}

	// Validate duplicate nonce
	if _, submitted := session.Submissions[clientNonce]; submitted {
		SendErrorResponse(conn, req.ID, "Duplicate submission")
		return
	}

	// Validate rate limit
	if time.Since(session.LastSubmit) < time.Second {
		SendErrorResponse(conn, req.ID, "Submission too frequent")
		return
	}

	expectedHash := calculateSHA256(session.ServerNonce + clientNonce)
	if expectedHash != result {
		SendErrorResponse(conn, req.ID, "Invalid result")
		return
	}

	// Mark submission as processed
	session.Submissions[clientNonce] = true
	session.LastSubmit = time.Now()
	// Update statistics after successful submission
	_ = session.StoreSuccSubmission()

	// Send success response
	SendSuccessResponse(conn, req.ID)
	logger.Info("Client %v submitted job %d with nonce %s", conn.RemoteAddr(), jobID, clientNonce)
}

func calculateSHA256(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

func GenerateServerNonce() string {
	return fmt.Sprintf("%d", rand.Int63())
}

func SendErrorResponse(conn net.Conn, id *int, errorMsg string) {
	response := Response{
		ID:     id,
		Result: false,
		Error:  errorMsg,
	}
	data, _ := json.Marshal(response)
	_, _ = conn.Write(append(data, '\n'))
}
func SendSuccessResponse(conn net.Conn, id *int) {
	response := Response{
		ID:     id,
		Result: true,
	}
	data, _ := json.Marshal(response)
	_, _ = conn.Write(append(data, '\n'))
}

func (s *Server) StartTaskDistribution(interval time.Duration, times int) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	if times > 0 {
		for i := 0; i < times; i++ {
			<-ticker.C
			s.mu.Lock()
			for conn, session := range s.sessions {
				s.DistributionJob(conn, session)
			}
			s.mu.Unlock()
		}
		logger.Info("Task distribution completed.")
		return
	}
	for range ticker.C {
		s.mu.Lock()
		for conn, session := range s.sessions {
			s.DistributionJob(conn, session)
		}
		s.mu.Unlock()
	}
}

func (s *Server) DistributionJob(conn net.Conn, session *Session) {
	session.mu.Lock()
	defer session.mu.Unlock()

	session.CurrJobID++
	session.ServerNonce = GenerateServerNonce()

	session.GetJob()
	session.CleanExpireJobHistory(100)

	task := Request{
		Method: "job",
		Params: map[string]interface{}{
			"job_id":       session.CurrJobID,
			"server_nonce": session.ServerNonce,
		},
	}

	message, _ := json.Marshal(task)
	_, err := conn.Write(append(message, '\n'))
	if err != nil {
		logger.Error("Failed to send job to client:", err) // set client ill
		delete(s.sessions, conn)
		conn.Close()
	}
}
func (s *Server) DistributionToForTest(userName string) {
	s.mu.Lock()
	for conn, session := range s.sessions {
		if session.Username == userName {
			s.DistributionJob(conn, session)
		}
	}
	s.mu.Unlock()
}
