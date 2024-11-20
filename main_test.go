package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestEventsHandler(t *testing.T) {
	var (
		tests = []struct {
			name    string
			workers int
		}{
			{
				name:    "single_worker",
				workers: 1,
			},

			{
				name:    "multiple_workers",
				workers: 3,
			},
		}
		expectedEvents = []*Event{
			{
				OrderType:  "Purchase",
				SessionID:  "29827525-06c9-4b1e-9d9b-7c4584e82f56",
				Card:       "4433**1409",
				EventDate:  "2023-01-04 13:44:52.835626 +00:00",
				WebsiteURL: "https://amazon.com",
			},
			{
				OrderType:  "CardVerify",
				SessionID:  "500cf308-e666-4639-aa9f-f6376015d1b4",
				Card:       "4433**1409",
				EventDate:  "2023-04-07 05:29:54.362216 +00:00",
				WebsiteURL: "https://adidas.com",
			},
			{
				OrderType:  "SendOtp",
				SessionID:  "500cf308-e666-4639-aa9f-f6376015d1b4",
				Card:       "4433**1409",
				EventDate:  "2023-04-06 22:52:34.930150 +00:00",
				WebsiteURL: "https://somon.tj",
			},
		}
	)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				notifiedEvents   = make([]*Event, 0)
				notifiedEventsMu = sync.Mutex{}
			)

			h := newEventsHandler(
				func(e *Event) {
					notifiedEventsMu.Lock()
					defer notifiedEventsMu.Unlock()
					notifiedEvents = append(notifiedEvents, e)
				})

			defer h.stopWorkers()

			h.startWorkers(tt.workers)

			server := httptest.NewServer(h)
			defer server.Close()

			for _, event := range expectedEvents {

				eventBytes, _ := json.Marshal(event)
				resp, err := http.Post(server.URL, "application/json", bytes.NewReader(eventBytes))
				if err != nil {
					t.Fatalf("http Post: %v", err)
				}
				if resp.StatusCode != http.StatusCreated {
					t.Errorf("expected status %d, got %d", http.StatusCreated, resp.StatusCode)
				}
				resp.Body.Close()
			}
			fmt.Println(expectedEvents, notifiedEvents, h.storage)
			if !equivalent(expectedEvents, notifiedEvents) {
				t.Errorf("notified events didn't match")
			}
			if !equivalent(expectedEvents, h.storage) {
				t.Errorf("storage events didn't match")
			}
		})
	}
}

func equivalent(expected, actual []*Event) bool {
	set := make(map[Event]struct{})

	for _, e := range expected {
		set[*e] = struct{}{}
	}
	for _, e := range actual {
		if _, ok := set[*e]; !ok {
			return false
		}
		delete(set, *e)
	}
	return len(set) == 0
}
