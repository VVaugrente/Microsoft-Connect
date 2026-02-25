package services

import (
	"sync"
	"time"
)

type ConversationMessage struct {
	Role    string
	Content string
	Time    time.Time
}

type ConversationStore struct {
	conversations map[string][]ConversationMessage
	mu            sync.RWMutex
	maxMessages   int
	ttl           time.Duration
	stopChan      chan struct{}
}

func NewConversationStore() *ConversationStore {
	store := &ConversationStore{
		conversations: make(map[string][]ConversationMessage),
		maxMessages:   50,
		ttl:           30 * time.Minute,
		stopChan:      make(chan struct{}),
	}
	go store.cleanup()
	return store
}

func (s *ConversationStore) Stop() {
	close(s.stopChan)
}

func (s *ConversationStore) AddMessage(conversationID, role, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	msg := ConversationMessage{
		Role:    role,
		Content: content,
		Time:    time.Now(),
	}

	s.conversations[conversationID] = append(s.conversations[conversationID], msg)

	// Limiter le nombre de messages
	if len(s.conversations[conversationID]) > s.maxMessages {
		s.conversations[conversationID] = s.conversations[conversationID][1:]
	}
}

func (s *ConversationStore) GetHistory(conversationID string) []ConversationMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.conversations[conversationID]
}

func (s *ConversationStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for id, messages := range s.conversations {
				if len(messages) > 0 && now.Sub(messages[len(messages)-1].Time) > s.ttl {
					delete(s.conversations, id)
				}
			}
			s.mu.Unlock()
		case <-s.stopChan:
			return
		}
	}
}
