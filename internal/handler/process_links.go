package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"go.uber.org/zap"

	"link-service/internal/domain"
	"link-service/internal/service"
)

type processLinksRequest struct {
	Links []string `json:"links"`
}

func ProcessLinks(serverCtx context.Context, srv *service.Service, requestTimeout time.Duration, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestCtx, cancel := context.WithTimeout(r.Context(), requestTimeout)
		defer cancel()

		var reqLinks processLinksRequest
		err := json.NewDecoder(r.Body).Decode(&reqLinks)
		if err != nil {
			http.Error(w, "cannot decode body", http.StatusBadRequest)
			logger.Warn("cannot decode body", zap.Error(err))
			return
		}

		rec, err := srv.Process(serverCtx, requestCtx, reqLinks.Links)
		if err != nil {
			if errors.Is(err, service.ErrAppStopped) {
				err = writeResponse(w, rec, logger)
				return
			}

			http.Error(w, "failed to process links", http.StatusInternalServerError)
			logger.Error("failed to process links", zap.Error(err))
			return
		}

		err = writeResponse(w, rec, logger)
	}
}

func writeResponse(w http.ResponseWriter, rec *domain.Record, logger *zap.Logger) error {
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(w).Encode(rec)
	if err != nil {
		logger.Warn("failed to encode response", zap.Error(err))
	}

	return err
}

//pdf := gofpdf.New("P", "mm", "A4", "")
//pdf.AddPage()
//pdf.SetFont("Arial", "", 14)
//
//for _, recID := range ids {
//rec := storage.Get(recID)
//for link, status := range rec.Results {
//pdf.Cell(0, 10, fmt.Sprintf("%s: %s", link, status))
//pdf.Ln(10)
//}
//}
//
//pdf.Output(w)
