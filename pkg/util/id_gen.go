package util

import (
	"sync/atomic"
)

var idCounter int64

func GenerateID() *int {
	id := atomic.AddInt64(&idCounter, 1)
	idInt := int(id)
	return &idInt
}
