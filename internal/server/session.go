package server

import (
	"fmt"
	"sync"
	"time"

	"luxor.tech/tcp_msg_processing_test/pkg/logger"
	rds_db "luxor.tech/tcp_msg_processing_test/rds-db"
)

// Session for client
type Session struct {
	Username string

	CurrJobID   int
	ServerNonce string
	Submissions map[string]bool // can extract to service/cache
	LastSubmit  time.Time

	JobHistory []TaskHistory

	mu sync.Mutex
}

func (s *Session) StoreSuccSubmission() interface{} {
	// incre submission count: can store in cache, async to db, could be bottle snake
	now := time.Now().UTC().Truncate(time.Minute)
	query := `
		INSERT INTO submissions (username, timestamp, submission_count)
		VALUES ($1, $2, 1)
		ON CONFLICT (username, timestamp)
		DO UPDATE SET submission_count = submissions.submission_count + 1;
	`
	_, err := rds_db.GetDb().Exec(query, s.Username, now)
	if err != nil {
		return fmt.Errorf("failed to update statistics for user %s: %v", s.Username, err)
	}
	logger.Info("Updated statistics for user %s", s.Username)
	return nil
}

// GetJob can flash to mq
func (s *Session) GetJob() {
	s.JobHistory = append(s.JobHistory, TaskHistory{
		JobID:       s.CurrJobID,
		ServerNonce: s.ServerNonce,
	})
}
func (s *Session) CleanExpireJobHistory(maxLength int) {
	if len(s.JobHistory) > maxLength {
		s.JobHistory = s.JobHistory[len(s.JobHistory)-maxLength:]
	}
}

func NewSession() *Session {
	return &Session{
		JobHistory:  make([]TaskHistory, 0),
		Submissions: make(map[string]bool),
	}
}
