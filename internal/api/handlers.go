package api

import (
	"encoding/json"
	"net/http"
	"project/internal/usecase"

	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	createOrderUC *usecase.CreateOrder
	getOrderUC    *usecase.GetOrder
	getWorkflowUC *usecase.GetWorkflow
	refundOrderUC *usecase.RefundOrder
}

func NewHandlers(createOrderUC *usecase.CreateOrder, getOrderUC *usecase.GetOrder, getWorkflowUC *usecase.GetWorkflow, refundOrderUC *usecase.RefundOrder) *Handlers {
	return &Handlers{
		createOrderUC: createOrderUC,
		getOrderUC:    getOrderUC,
		getWorkflowUC: getWorkflowUC,
		refundOrderUC: refundOrderUC,
	}
}

func (h *Handlers) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID  string  `json:"user_id"`
		Amount  float64 `json:"amount"`
		From    string  `json:"from"`
		To      string  `json:"to"`
		Date    string  `json:"date"`
		Time    string  `json:"time"`
		Airline string  `json:"airline"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	params := usecase.CreateOrderParams{
		UserID:  req.UserID,
		Amount:  req.Amount,
		From:    req.From,
		To:      req.To,
		Date:    req.Date,
		Time:    req.Time,
		Airline: req.Airline,
	}

	id, err := h.createOrderUC.Execute(r.Context(), params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "CREATED",
		"order_id": id,
	})
}

func (h *Handlers) GetOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing order id", http.StatusBadRequest)
		return
	}

	order, err := h.getOrderUC.Execute(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	json.NewEncoder(w).Encode(order)
}

func (h *Handlers) GetWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing order id", http.StatusBadRequest)
		return
	}

	workflow, err := h.getWorkflowUC.Execute(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	json.NewEncoder(w).Encode(workflow)
}

func (h *Handlers) RefundOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing order id", http.StatusBadRequest)
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// optional body, ignoring error if empty or valid json required?
		// for simplicity let's allow empty reason
	}

	params := usecase.RefundOrderParams{
		OrderID: id,
		Reason:  req.Reason,
	}

	if err := h.refundOrderUC.Execute(r.Context(), params); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "refund_initiated"})
}
