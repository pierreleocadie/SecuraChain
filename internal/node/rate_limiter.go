package node

import (
	"sync"
	"time"
)

// RateLimiter control the rate of messages from a sender.
type RateLimiter struct {
	messages map[string][]time.Time
	mu       sync.Mutex
	limit    int
	interval time.Duration
}

func NewRateLimiter(limit int, interval time.Duration) *RateLimiter {
	return &RateLimiter{
		messages: make(map[string][]time.Time),
		limit:    limit,
		interval: interval,
	}
}

// Allow checks if the sender is allowed to send a message.
func (rl *RateLimiter) Allow(sender string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	messages, exists := rl.messages[sender]

	// Clean up old messages.
	if exists {
		var filteredMessages []time.Time
		for _, msgTime := range messages {
			if now.Sub(msgTime) <= rl.interval {
				filteredMessages = append(filteredMessages, msgTime)
			}
		}
		rl.messages[sender] = filteredMessages
	}

	// Checks if the sender has reached the limit.
	if len(rl.messages[sender]) >= rl.limit {
		return false
	}

	// Add the new message.
	rl.messages[sender] = append(rl.messages[sender], now)
	return true
}
