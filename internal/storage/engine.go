package storage

import (
	"fmt"
	"gosql-db/internal/storage/compaction"
	"gosql-db/internal/storage/memtable"
	"gosql-db/internal/storage/sstable"
	"gosql-db/internal/storage/wal"
	"sync"
)

// Engine является главным интерфейсом движка хранения данных.
type Engine struct {
	mu        sync.RWMutex
	memTable  *memtable.Table
	wal       *wal.Log
	sstables  []*sstable.Table
	config    Config
	compactor *compaction.Worker
}

// Config содержит параметры конфигурации движка.
type Config struct {
	DataDir         string
	WALDir          string
	MaxMemTableSize int64
}

// NewEngine создает новый экземпляр движка хранения.
func NewEngine(cfg Config) (*Engine, error) {
	// Инициализация WAL
	walLog, err := wal.NewLog(cfg.WALDir)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания WAL: %w", err)
	}

	// Инициализация MemTable
	mt := memtable.NewTable(cfg.MaxMemTableSize)

	engine := &Engine{
		memTable: mt,
		wal:      walLog,
		config:   cfg,
	}

	// Восстановление состояния из WAL и SSTable при старте
	if err := engine.recoverState(); err != nil {
		return nil, fmt.Errorf("ошибка восстановления состояния: %w", err)
	}

	// Запуск воркера компактификации
	engine.compactor = compaction.NewWorker(engine)
	go engine.compactor.Run()

	return engine, nil
}

// Put записывает ключ-значение в базу данных.
func (e *Engine) Put(key []byte, value []byte) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Запись в WAL для долговечности
	if err := e.wal.Append(key, value); err != nil {
		return fmt.Errorf("ошибка записи в WAL: %w", err)
	}

	// Запись в MemTable
	inserted, err := e.memTable.Put(key, value)
	if err != nil {
		return fmt.Errorf("ошибка записи в MemTable: %w", err)
	}

	// Проверка необходимости сброса (Flush) на диск
	if inserted && e.memTable.Size() >= e.config.MaxMemTableSize {
		if err := e.flushMemTable(); err != nil {
			return fmt.Errorf("ошибка сброса MemTable: %w", err)
		}
	}

	return nil
}

// Get retrieves значение по ключу.
func (e *Engine) Get(key []byte) ([]byte, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Поиск в MemTable
	val, found := e.memTable.Get(key)
	if found {
		return val, nil
	}

	// Поиск в SSTables (в обратном порядке, от новых к старым)
	for i := len(e.sstables) - 1; i >= 0; i-- {
		val, found := e.sstables[i].Get(key)
		if found {
			return val, nil
		}
	}

	return nil, fmt.Errorf("ключ не найден")
}

// Delete удаляет ключ из базы данных.
func (e *Engine) Delete(key []byte) error {
	// Реализация удаления через маркер tombstone
	return e.Put(key, nil) 
}

// Close корректно закрывает все ресурсы движка.
func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if err := e.flushMemTable(); err != nil {
		return err
	}
	
	if err := e.wal.Close(); err != nil {
		return err
	}

	e.compactor.Stop()
	return nil
}

// flushMemTable сбрасывает содержимое памяти на диск в виде SSTable.
func (e *Engine) flushMemTable() error {
	// Логика сериализации MemTable в SSTable
	// Упрощено для примера структуры
	sst, err := sstable.CreateTableFromMemTable(e.memTable, e.config.DataDir)
	if err != nil {
		return err
	}
	e.sstables = append(e.sstables, sst)
	e.memTable = memtable.NewTable(e.config.MaxMemTableSize)
	
	// Очистка WAL после успешного сброса
	return e.wal.Truncate()
}

// recoverState восстанавливает данные при старте.
func (e *Engine) recoverState() error {
	// 1. Загрузка списка SSTable
	// 2. Проигрывание WAL
	return nil 
}