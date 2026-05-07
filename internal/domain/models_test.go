package domain

import (
	"testing"
	"time"
)

func TestMessage(t *testing.T) {
	msg := Message{
		ID:        1,
		Date:      time.Now(),
		Views:     100,
		Forwards:  5,
		Reactions: map[string]int{"👍": 10},
	}
	if msg.ID != 1 {
		t.Errorf("ID = %d; want 1", msg.ID)
	}
}

func TestChatInfo(t *testing.T) {
	ci := ChatInfo{
		ID:          "@test",
		Subscribers: 500,
		Description: "desc",
	}
	if ci.Subscribers != 500 {
		t.Errorf("Subscribers = %d; want 500", ci.Subscribers)
	}
}

func TestAnalyticsSummary(t *testing.T) {
    b := true
	as := AnalyticsSummary{
		AverageViews:      100.5,
		EngagementRate:    5.2,
		 ContainsKeyword:   &b,
	}
	if as.EngagementRate != 5.2 {
		t.Errorf("EngagementRate = %f; want 5.2", as.EngagementRate)
	}
}
