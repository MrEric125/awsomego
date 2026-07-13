package adapter

import (
	"crypto/rand"
	"fmt"
	"time"
)

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}
