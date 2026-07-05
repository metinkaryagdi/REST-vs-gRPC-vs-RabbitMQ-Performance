package main

import (
	"encoding/json"
	"io"
	"net/http"
)

type EventPayload struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Merchant string  `json:"merchant"`
	Location string  `json:"location"`
	Device   string  `json:"device"`
}

type LogRequest struct {
	EventID   string        `json:"event_id"`
	Timestamp int64         `json:"timestamp"`
	UserID    string        `json:"user_id"`
	EventType string        `json:"event_type"`
	Payload   *EventPayload `json:"payload"`
}

type LogResponse struct {
	Status string `json:"status"`
}

func restLogHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`{"error":"Method not allowed"}`))
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Failed to read body"}`))
		return
	}
	defer r.Body.Close()

	var req LogRequest
	if err := json.Unmarshal(body, &req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Failed to unmarshal JSON"}`))
		return
	}

	if req.EventID == "" || req.UserID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Missing fields"}`))
		return
	}

	resp := LogResponse{Status: "success"}
	respBytes, _ := json.Marshal(resp)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respBytes)
}

func StartRESTServer(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/log", restLogHandler)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return server
}
