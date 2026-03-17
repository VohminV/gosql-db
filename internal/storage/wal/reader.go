package wal

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// Entry представляет восстановленную запись из WAL.
type Entry struct {
	Key   []byte
	Value []byte
}

// Replay читает все сегменты WAL начиная с указанного ID и возвращает список записей для восстановления состояния.
func Replay(dir string, startID uint64) ([]Entry, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения директории WAL: %w", err)
	}

	var segmentPaths []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		id, err := parseSegmentName(file.Name())
		if err != nil || id < startID {
			continue
		}
		segmentPaths = append(segmentPaths, filepath.Join(dir, file.Name()))
	}

	// Сортировка сегментов по возрастанию ID для корректного порядка воспроизведения
	sort.Strings(segmentPaths)

	var entries []Entry

	for _, path := range segmentPaths {
		seg, err := OpenSegment(path, false) // Открываем для чтения
		if err != nil {
			return nil, fmt.Errorf("ошибка открытия сегмента %s: %w", path, err)
		}

		segEntries, err := readSegmentEntries(seg.file)
		seg.Close()
		
		if err != nil {
			return nil, fmt.Errorf("ошибка чтения записей из сегмента %s: %w", path, err)
		}

		entries = append(entries, segEntries...)
	}

	return entries, nil
}

// readSegmentEntries читает все записи из одного файла сегмента.
func readSegmentEntries(r io.Reader) ([]Entry, error) {
	var entries []Entry
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	offset := 0
	for offset < len(data) {
		// Чтение длины ключа
		if offset+4 > len(data) {
			break // Неполная запись, игнорируем (возможно, обрыв при крахе)
		}
		keyLen := binary.BigEndian.Uint32(data[offset : offset+4])
		offset += 4

		// Чтение ключа
		if offset+int(keyLen) > len(data) {
			break
		}
		key := data[offset : offset+int(keyLen)]
		offset += int(keyLen)

		// Чтение длины значения
		if offset+4 > len(data) {
			break
		}
		valLen := binary.BigEndian.Uint32(data[offset : offset+4])
		offset += 4

		// Чтение значения
		if offset+int(valLen) > len(data) {
			break
		}
		value := data[offset : offset+int(valLen)]
		offset += int(valLen)

		entries = append(entries, Entry{Key: key, Value: value})
	}

	return entries, nil
}