package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go.uber.org/zap"

	"github.com/jung-kurt/gofpdf"

	"link-service/internal/repository"
)

type getLinksRequest struct {
	LinksList []int64 `json:"links_list"`
}

func GetLinks(repo repository.Repository, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var reqLinks getLinksRequest
		err := json.NewDecoder(r.Body).Decode(&reqLinks)
		if err != nil {
			http.Error(w, "cannot decode body", http.StatusBadRequest)
			logger.Warn("cannot decode body", zap.Error(err))
			return
		}

		pdf := gofpdf.New("P", "mm", "A4", "")
		pdf.AddPage()
		pdf.SetFont("Arial", "", 12)

		for _, id := range reqLinks.LinksList {
			rec, err := repo.GetRecord(id)
			if err != nil {
				logger.Warn("failed to get record", zap.Int64("id", id), zap.Error(err))
				continue
			}

			pdf.CellFormat(0, 8, "Record: "+strconv.FormatInt(rec.ID, 10), "", 1, "", false, 0, "")
			for link, status := range rec.Links {
				pdf.CellFormat(0, 6, link+": "+status, "", 1, "", false, 0, "")
			}

			pdf.Ln(4)
		}

		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", "attachment; filename=records.pdf")

		err = pdf.Output(w)
		if err != nil {
			logger.Error("failed to write pdf", zap.Error(err))
		}
	}
}
