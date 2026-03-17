package wal

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const segmentPrefix = "wal_"

// Segment представляет отдельный файл журнала.
type Segment struct {
	file *os.File
	size int64
}

// CreateSegment создает новый файл сегмента.
func CreateSegment(path string) (*Segment, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать файл сегмента: %w", err)
	}
	
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	return &Segment{file: f, size: info.Size()}, nil
}

// OpenSegment открывает существующий файл сегмента.
// appendMode=true означает открытие для дозаписи, false - для чтения.
func OpenSegment(path string, appendMode bool) (*Segment, error) {
	flag := os.O_RDONLY
	if appendMode {
		flag = os.O_RDWR | os.O_APPEND
	}

	f, err := os.OpenFile(path, flag, 0644)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл сегмента: %w", err)
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	return &Segment{file: f, size: info.Size()}, nil
}

// Write записывает данные в сегмент и выполняет синхронизацию с диском (fsync).
func (s *Segment) Write(data []byte) error {
	n, err := s.file.Write(data)
	if err != nil {
		return err
	}
	s.size += int64(n)

	// Критически важно: гарантировать физическую запись на диск
	if err := s.file.Sync(); err != nil {
		return fmt.Errorf("ошибка синхронизации сегмента с диском: %w", err)
	}

	return nil
}

// Size возвращает текущий размер заполненных данных в сегменте.
func (s *Segment) Size() int64 {
	return s.size
}

// Close закрывает файл сегмента.
func (s *Segment) Close() error {
	if s.file != nil {
		return s.file.Close()
	}
	return nil
}

// formatSegmentName генерирует имя файла сегмента на основе ID.
func formatSegmentName(id uint64) string {
	return fmt.Sprintf("%s%020d", segmentPrefix, id)
}

// parseSegmentName извлекает ID из имени файла.
func parseSegmentName(name string) (uint64, error) {
	if !strings.HasPrefix(name, segmentPrefix) {
		return 0, fmt.Errorf("неверный префикс имени сегмента")
	}
	numStr := strings.TrimPrefix(name, segmentPrefix)
	id, err := strconv.ParseUint(numStr, 10, 64)
	if err != nil {
		return 0, err
	}
	return id, nil
}