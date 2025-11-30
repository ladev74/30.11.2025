package repository

import "link-service/internal/domain"

type Repository interface {
	SaveRecord(record *domain.Record) error
	SaveTempRecord(record *domain.Record) error
	LoadTempRecords() ([]domain.Record, error)
	GetRecord(id int64) (*domain.Record, error)
	ClearTempFile() error
	LoadLastLinksNum() int64
}
