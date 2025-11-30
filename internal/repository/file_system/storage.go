package filesystem

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"

	"link-service/internal/domain"
)

type Config struct {
	DirPath      string `env:"STORAGE_DIR_PATH" env-required:"true"`
	FileName     string `env:"STORAGE_FILE_NAME" env-required:"true"`
	TempFileName string `env:"STORAGE_TEMP_FILE_NAME" env-required:"true"`
}

type Storage struct {
	mu       *sync.Mutex
	path     string
	tempPath string
	logger   *zap.Logger
}

func New(cfg *Config, logger *zap.Logger) (*Storage, error) {
	err := os.MkdirAll(cfg.DirPath, 0755)
	if err != nil {
		logger.Error("failed to create dir", zap.String("dir_path", cfg.DirPath), zap.Error(err))
		return nil, fmt.Errorf("failed to create dir: %s: %w", cfg.DirPath, err)
	}

	filePath := filepath.Join(cfg.DirPath, cfg.FileName)
	tempFilePath := filepath.Join(cfg.DirPath, cfg.TempFileName)

	file, err := os.OpenFile(filePath,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)
	if err != nil {
		logger.Error("failed to create file", zap.String("file_name", cfg.FileName), zap.Error(err))
		return nil, fmt.Errorf("failed to create file: %s: %w", cfg.FileName, err)
	}

	defer file.Close()

	tempFile, err := os.OpenFile(tempFilePath,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)
	if err != nil {
		logger.Error("failed to create temp file", zap.String("file_name", cfg.TempFileName), zap.Error(err))
		return nil, fmt.Errorf("failed to create temp file: %s: %w", cfg.TempFileName, err)
	}

	defer tempFile.Close()

	logger.Info("files created", zap.String("file", filePath), zap.String("temp_path", tempFilePath))

	return &Storage{
		mu:       &sync.Mutex{},
		path:     filePath,
		tempPath: tempFilePath,
		logger:   logger,
	}, nil
}

func (s *Storage) SaveRecord(record *domain.Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.OpenFile(s.path, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		s.logger.Error("failed to open file", zap.String("path", s.path), zap.Error(err))
		return fmt.Errorf("failed to open file: %s: %w", s.path, err)
	}
	defer file.Close()

	data, err := json.Marshal(record)
	if err != nil {
		s.logger.Error("failed to marshal record", zap.Error(err))
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	data = append(data, '\n')

	_, err = file.Write(data)
	if err != nil {
		s.logger.Error("failed to write record", zap.Error(err))
		return fmt.Errorf("failed to write record: %w", err)
	}

	s.logger.Info("successfully wrote record")
	return nil
}

func (s *Storage) SaveTempRecord(record *domain.Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tempFile, err := os.OpenFile(s.tempPath, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		s.logger.Error("failed to open temp file", zap.String("path", s.tempPath), zap.Error(err))
		return fmt.Errorf("failed to open temp file: %s: %w", s.tempPath, err)
	}
	defer tempFile.Close()

	data, err := json.Marshal(record)
	if err != nil {
		s.logger.Error("failed to marshal temp record", zap.Error(err))
		return fmt.Errorf("failed to marshal temp record: %w", err)
	}

	data = append(data, '\n')

	_, err = tempFile.Write(data)
	if err != nil {
		s.logger.Error("failed to write temp record", zap.Error(err))
		return fmt.Errorf("failed to write temp record: %w", err)
	}

	s.logger.Info("successfully wrote temp record")
	return nil
}

func (s *Storage) LoadTempRecords() ([]domain.Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tempFile, err := os.Open(s.tempPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open temp file: %w", err)
	}
	defer tempFile.Close()

	var records []domain.Record
	decoder := json.NewDecoder(tempFile)

	for {
		var rec domain.Record
		err = decoder.Decode(&rec)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return nil, fmt.Errorf("failed to decode temp record: %w", err)
		}

		records = append(records, rec)
	}

	return records, nil
}

func (s *Storage) GetRecord(id int64) (*domain.Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.Open(s.path)
	if err != nil {
		s.logger.Error("failed to open file", zap.String("path", s.path), zap.Error(err))
		return nil, fmt.Errorf("failed to open file: %s: %w", s.path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var rec domain.Record
		err = json.Unmarshal(scanner.Bytes(), &rec)
		if err != nil {
			continue
		}

		if rec.ID == id {
			return &rec, nil
		}
	}

	err = scanner.Err()
	if err != nil {
		s.logger.Error("failed to scan file", zap.String("path", s.path), zap.Error(err))
		return nil, fmt.Errorf("failed to scan file: %s: %w", s.path, err)
	}

	return nil, fmt.Errorf("record with ID %d not found", id)
}

func (s *Storage) ClearTempFile() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := os.WriteFile(s.tempPath, []byte{}, 0644)

	return err
}

func (s *Storage) LoadLastLinksNum() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.Open(s.path)
	if err != nil {
		s.logger.Error("failed to open file", zap.String("path", s.path), zap.Error(err))
		return 0
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		s.logger.Error("failed to stat file", zap.Error(err))
		return 0
	}

	if stat.Size() == 0 {
		return 0
	}

	buf := make([]byte, 1)
	var lastLine []byte
	pos := stat.Size() - 1

	for pos >= 0 {
		_, err = file.ReadAt(buf, pos)
		if err != nil {
			s.logger.Error("failed to read file", zap.Error(err))
			return 0
		}

		if buf[0] == '\n' && pos != stat.Size()-1 {
			break
		}

		lastLine = append([]byte{buf[0]}, lastLine...)
		pos--
	}

	var rec domain.Record
	err = json.Unmarshal(lastLine, &rec)
	if err != nil {
		s.logger.Error("failed to unmarshal last record", zap.Error(err))
		return 0
	}

	return rec.ID
}
