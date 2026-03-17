package transaction

import (
	"fmt"
	"sync/atomic"
	"time"
)

// Status определяет состояние транзакции.
type Status int

const (
	StatusActive Status = iota
	StatusCommitted
	StatusAborted
)

// Transaction представляет объект транзакции с поддержкой MVCC.
type Transaction struct {
	ID        uint64
	StartTime time.Time
	Status    Status
	Snapshot  *Snapshot
	engine    StorageEngine // Интерфейс к хранилищу
	writeSet  map[string][]byte
	readSet   map[string]uint64 // Key -> Version
}

// StorageEngine определяет минимальный интерфейс для транзакций.
type StorageEngine interface {
	Put(key, value []byte) error
	Get(key []byte) ([]byte, error)
}

var globalTxID uint64

// Begin начинает новую транзакцию.
func Begin(engine StorageEngine) (*Transaction, error) {
	id := atomic.AddUint64(&globalTxID, 1)
	
	snapshot, err := CreateSnapshot(engine, id)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания снимка: %w", err)
	}

	return &Transaction{
		ID:        id,
		StartTime: time.Now(),
		Status:    StatusActive,
		Snapshot:  snapshot,
		engine:    engine,
		writeSet:  make(map[string][]byte),
		readSet:   make(map[string]uint64),
	}, nil
}

// Get читает данные в контексте текущей транзакции.
func (t *Transaction) Get(key string) ([]byte, error) {
	if t.Status != StatusActive {
		return nil, fmt.Errorf("транзакция не активна")
	}

	// Проверка локальной записи
	if val, ok := t.writeSet[key]; ok {
		if val == nil {
			return nil, fmt.Errorf("ключ удален в текущей транзакции")
		}
		return val, nil
	}

	// Чтение из снимка (MVCC)
	val, version, err := t.Snapshot.Get(key)
	if err != nil {
		return nil, err
	}
	t.readSet[key] = version
	return val, nil
}

// Put записывает данные в буфер транзакции.
func (t *Transaction) Put(key string, value []byte) error {
	if t.Status != StatusActive {
		return fmt.Errorf("транзакция не активна")
	}
	t.writeSet[key] = value
	return nil
}

// Commit фиксирует изменения.
func (t *Transaction) Commit() error {
	if t.Status != StatusActive {
		return fmt.Errorf("невозможно зафиксировать неактивную транзакцию")
	}

	// Здесь должна быть логика проверки конфликтов сериализуемости (SSI)
	// и атомарной записи в WAL и MemTable

	t.Status = StatusCommitted
	return nil
}

// Rollback отменяет изменения.
func (t *Transaction) Rollback() error {
	if t.Status != StatusActive {
		return fmt.Errorf("транзакция уже завершена")
	}
	t.writeSet = make(map[string][]byte)
	t.Status = StatusAborted
	return nil
}