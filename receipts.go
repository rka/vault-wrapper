package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	receiptSessionCookie = "vault_wrapper_sender"
	receiptRetention     = time.Hour
	receiptHeartbeat     = 15 * time.Second
	maxSessionStreams    = 2
	maxSessionReceipts   = 100
	maxTrackedReceipts   = 10000
)

type receiptEvent struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expires_at"`
}

type receiptRecord struct {
	receiptEvent
	SessionID string
}

type receiptHub struct {
	mu          sync.Mutex
	receipts    map[string]receiptRecord
	subscribers map[string]map[chan receiptEvent]struct{}
}

var liveReceipts = &receiptHub{
	receipts:    make(map[string]receiptRecord),
	subscribers: make(map[string]map[chan receiptEvent]struct{}),
}

func newOpaqueID() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func validOpaqueID(value string) bool {
	if len(value) != 43 {
		return false
	}
	raw, err := base64.RawURLEncoding.DecodeString(value)
	return err == nil && len(raw) == 32
}

func senderSessionForWrap(r *http.Request) (string, bool, error) {
	if cookie, err := r.Cookie(receiptSessionCookie); err == nil && validOpaqueID(cookie.Value) {
		return cookie.Value, false, nil
	}
	id, err := newOpaqueID()
	return id, true, err
}

func setSenderSessionCookie(w http.ResponseWriter, r *http.Request, sessionID string) {
	secure := r.TLS != nil
	if TrustProxyHeaders {
		proto := strings.TrimSpace(strings.SplitN(r.Header.Get("X-Forwarded-Proto"), ",", 2)[0])
		secure = secure || strings.EqualFold(proto, "https")
	}
	http.SetCookie(w, &http.Cookie{
		Name:     receiptSessionCookie,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
}

func senderSessionFromRequest(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(receiptSessionCookie)
	if err != nil || !validOpaqueID(cookie.Value) {
		return "", false
	}
	return cookie.Value, true
}

func (h *receiptHub) add(sessionID, receiptID string, expiresAt time.Time) (receiptEvent, bool) {
	event := receiptEvent{ID: receiptID, Status: "waiting", ExpiresAt: expiresAt.UTC()}
	h.mu.Lock()
	if len(h.receipts) >= maxTrackedReceipts {
		h.mu.Unlock()
		return event, false
	}
	sessionReceipts := 0
	for _, record := range h.receipts {
		if record.SessionID == sessionID {
			sessionReceipts++
		}
	}
	if sessionReceipts >= maxSessionReceipts {
		h.mu.Unlock()
		return event, false
	}
	h.receipts[receiptID] = receiptRecord{receiptEvent: event, SessionID: sessionID}
	h.publishLocked(sessionID, event)
	h.mu.Unlock()

	expiryDelay := time.Until(expiresAt)
	if expiryDelay < 0 {
		expiryDelay = 0
	}
	time.AfterFunc(expiryDelay, func() { h.expire(receiptID) })
	time.AfterFunc(expiryDelay+receiptRetention, func() { h.remove(receiptID) })
	return event, true
}

func (h *receiptHub) markRetrieved(receiptID string) bool {
	if receiptID == "" {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	record, ok := h.receipts[receiptID]
	if !ok || record.Status != "waiting" {
		return false
	}
	record.Status = "retrieved"
	h.receipts[receiptID] = record
	h.publishLocked(record.SessionID, record.receiptEvent)
	time.AfterFunc(receiptRetention, func() { h.remove(receiptID) })
	return true
}

func (h *receiptHub) expire(receiptID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	record, ok := h.receipts[receiptID]
	if !ok || record.Status != "waiting" {
		return
	}
	record.Status = "expired"
	h.receipts[receiptID] = record
	h.publishLocked(record.SessionID, record.receiptEvent)
}

func (h *receiptHub) remove(receiptID string) {
	h.mu.Lock()
	delete(h.receipts, receiptID)
	h.mu.Unlock()
}

func (h *receiptHub) publishLocked(sessionID string, event receiptEvent) {
	for subscriber := range h.subscribers[sessionID] {
		select {
		case subscriber <- event:
		default:
		}
	}
}

func (h *receiptHub) subscribe(sessionID string) (<-chan receiptEvent, []receiptEvent, func(), bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.subscribers[sessionID]) >= maxSessionStreams {
		return nil, nil, nil, false
	}
	channel := make(chan receiptEvent, 16)
	if h.subscribers[sessionID] == nil {
		h.subscribers[sessionID] = make(map[chan receiptEvent]struct{})
	}
	h.subscribers[sessionID][channel] = struct{}{}

	snapshot := make([]receiptEvent, 0)
	for _, record := range h.receipts {
		if record.SessionID == sessionID {
			snapshot = append(snapshot, record.receiptEvent)
		}
	}
	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			h.mu.Lock()
			delete(h.subscribers[sessionID], channel)
			if len(h.subscribers[sessionID]) == 0 {
				delete(h.subscribers, sessionID)
			}
			h.mu.Unlock()
		})
	}
	return channel, snapshot, unsubscribe, true
}

func writeReceiptEvent(w http.ResponseWriter, flusher http.Flusher, event receiptEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "event: receipt\ndata: %s\n\n", payload); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}

func receiptEventsHandler(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodGet) {
		return
	}
	sessionID, ok := senderSessionFromRequest(r)
	if !ok {
		http.Error(w, "No active sender session", http.StatusUnauthorized)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming is unavailable", http.StatusInternalServerError)
		return
	}
	events, snapshot, unsubscribe, ok := liveReceipts.subscribe(sessionID)
	if !ok {
		http.Error(w, "Too many live status connections", http.StatusTooManyRequests)
		return
	}
	defer unsubscribe()

	controller := http.NewResponseController(w)
	if err := controller.SetWriteDeadline(time.Time{}); err != nil && !errors.Is(err, http.ErrNotSupported) {
		log.Printf("WARN  [%s] receipts: could not disable stream write deadline: %v", reqID(r), err)
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprint(w, "retry: 3000\n\n"); err != nil {
		return
	}
	flusher.Flush()
	for _, event := range snapshot {
		if err := writeReceiptEvent(w, flusher, event); err != nil {
			return
		}
	}

	heartbeat := time.NewTicker(receiptHeartbeat)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-events:
			if err := writeReceiptEvent(w, flusher, event); err != nil {
				return
			}
		case <-heartbeat.C:
			if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
