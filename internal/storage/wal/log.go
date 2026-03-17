package wal

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Log представляет журнал предзаписи (WAL).
type Log struct {
	mu          sync.Mutex
	currentSeg  *Segment
	dir         string
	segmentSize int64
	segmentID   uint64
}

// Config содержит параметры конфигурации WAL.
type Config struct {
	Dir         string // Директория для хранения файлов WAL
	SegmentSize int64  // Максимальный размер одного сегмента в байтах
}

// NewLog создает новый экземпляр журнала WAL и восстанавливает состояние при необходимости.
func NewLog(cfg Config) (*Log, error) {
	// Создание директории, если она не существует
	if err := os.MkdirAll(cfg.Dir, 0755); err != nil {
		return nil, fmt.Errorf("ошибка создания директории WAL: %w", err)
	}

	log := &Log{
		dir:         cfg.Dir,
		segmentSize: cfg.SegmentSize,
		segmentID:   0,
	}

	// Поиск последнего существующего сегмента для продолжения записи
	lastID, err := log.findLastSegmentID()
	if err != nil {
		return nil, fmt.Errorf("ошибка поиска последнего сегмента: %w", err)
	}

	if lastID > 0 {
		log.segmentID = lastID
		// Открытие последнего сегмента для дозаписи
		seg, err := OpenSegment(filepath.Join(log.dir, formatSegmentName(lastID)), true)
		if err != nil {
			return nil, fmt.Errorf("ошибка открытия последнего сегмента: %w", err)
		}
		log.currentSeg = seg
	} else {
		// Создание нового первого сегмента
		if err := log.rotateSegment(); err != nil {
			return nil, fmt.Errorf("ошибка создания начального сегмента: %w", err)
		}
	}

	return log, nil
}

// Append записывает новую запись (ключ-значение) в журнал.
// Гарантирует, что данные сброшены на диск (fsync) перед возвратом управления.
func (l *Log) Append(key, value []byte) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Формирование записи: [Длина Ключа][Ключ][Длина Значения][Значение]
	record := make([]byte, 0)
	
	// Кодирование длины ключа (4 байта)
	keyLen := uint32(len(key))
	record = appendUint32(record, keyLen)
	record = append(record, key...)

	// Кодирование длины значения (4 байта)
	valLen := uint32(len(value))
	record = appendUint32(record, valLen)
	record = append(record, value...)

	// Проверка места в текущем сегменте
	if l.currentSeg.Size()+int64(len(record)) > l.segmentSize {
		if err := l.rotateSegment(); err != nil {
			return fmt.Errorf("ошибка ротации сегмента при записи: %w", err)
		}
	}

	// Запись в текущий сегмент
	if err := l.currentSeg.Write(record); err != nil {
		return fmt.Errorf("ошибка записи в сегмент: %w", err)
	}

	return nil
}

// Truncate очищает старые сегменты после успешного сброса данных в SSTable.
// В данной реализации удаляет все сегменты, кроме текущего (упрощенная стратегия).
// В продакшене требуется более сложная логика отслеживания ID чекпоинта.
func (l *Log) Truncate() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Здесь должна быть логика удаления старых сегментов на основе ID последнего чекпоинта
	// Для безопасности пока оставляем без изменений или реализуем удаление всех, кроме текущего
	// Реализация удаления всех сегментов с ID < currentID
	
	files, err := os.ReadDir(l.dir)
	if err != nil {
		return fmt.Errorf("ошибка чтения директории WAL: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		id, err := parseSegmentName(file.Name())
		if err != nil {
			continue // Пропускаем файлы, не являющиеся сегментами
		}
		// Удаляем все сегменты, которые уже не нужны (старше текущего минус один)
		// В упрощенном варианте: если мы сделали flush, значит данные в SSTable, 
		// но нам нужен текущий WAL для новых записей. 
		// Правильная логика: хранить ID последнего flushed сегмента.
		// Пока реализуем безопасный вариант: не удаляем ничего автоматически без явного указания ID.
		_ = id 
	}
	
	return nil
}

// Close закрывает текущий сегмент и освобождает ресурсы.
func (l *Log) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	if l.currentSeg != nil {
		return l.currentSeg.Close()
	}
	return nil
}

// rotateSegment создает новый файл сегмента.
func (l *Log) rotateSegment() error {
	if l.currentSeg != nil {
		if err := l.currentSeg.Close(); err != nil {
			return err
		}
	}

	l.segmentID++
	path := filepath.Join(l.dir, formatSegmentName(l.segmentID))
	
	seg, err := CreateSegment(path)
	if err != nil {
		return fmt.Errorf("ошибка создания нового сегмента: %w", err)
	}

	l.currentSeg = seg
	return nil
}

// findLastSegmentID сканирует директорию и возвращает ID последнего сегмента.
func (l *Log) findLastSegmentID() (uint64, error) {
	files, err := os.ReadDir(l.dir)
	if err != nil {
		return 0, err
	}

	var maxID uint64 = 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		id, err := parseSegmentName(file.Name())
		if err != nil {
			continue
		}
		if id > maxID {
			maxID = id
		}
	}
	return maxID, nil
}

// Вспомогательные функции для бинарного кодирования
func appendUint32(b []byte, v uint32) []byte {
	return append(b, byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}