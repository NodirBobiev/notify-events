package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"
)

type Event struct {
	OrderType  string `json:"orderType"`
	SessionID  string `json:"sessionId"`
	Card       string `json:"card"`
	EventDate  string `json:"eventDate"`
	WebsiteURL string `json:"websiteUrl"`
}

type eventsHandler struct {
	eventsQueue  chan *Event
	workersGroup sync.WaitGroup
	notify       func(*Event)

	storage   []*Event
	storageMu sync.Mutex
}

func newEventsHandler(notify func(*Event)) *eventsHandler {
	return &eventsHandler{
		eventsQueue:  make(chan *Event, 100),
		workersGroup: sync.WaitGroup{},
		notify:       notify,

		storage:   make([]*Event, 0),
		storageMu: sync.Mutex{},
	}
}

func (h *eventsHandler) store(e *Event) {
	h.storageMu.Lock()
	defer h.storageMu.Unlock()
	h.storage = append(h.storage, e)
}

func (h *eventsHandler) startWorkers(workers int) {
	h.workersGroup.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer h.workersGroup.Done()

			for e := range h.eventsQueue {
				h.notify(e)
			}
		}()
	}
}

func (h *eventsHandler) stopWorkers() {
	close(h.eventsQueue)
	h.workersGroup.Wait()
}

func (h *eventsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	event := &Event{}
	err := json.NewDecoder(r.Body).Decode(event)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.store(event)
	h.eventsQueue <- event

	w.WriteHeader(http.StatusCreated)
}

func main() {

	h := newEventsHandler(func(e *Event) { log.Println(e) })
	h.startWorkers(3)

	server := &http.Server{
		Addr:    ":8080",
		Handler: h,
	}

	go func() {
		log.Println("Server is running...")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
		log.Println("Server stopped")
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	<-sigChan

	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), time.Second)
	defer shutdownRelease()

	// we, first, wait until the server is completely shutdown
	// then we stop the workers to ensure no event left in the middle of the road :)

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Stopping workers...")
	h.stopWorkers()

	log.Println("Graceful shutdown complete")
}
