package storage

import (
	"fmt"
	"os"
	"sync"

	"gosql-db/internal/storage/compaction"
	"gosql-db/internal/storage/memtable"
	"gosql-db/internal/storage/sstable"
	"gosql-db/internal/storage/wal"
)

// Engine является главным интерфейсом движка хранения данных.
type Engine struct {
	mu        sync.RWMutex
	memTable  *memtable.Table
	wal       *wal.Log
	sstables  []*sstable.Table
	config    Config
	compactor *compaction.Worker
	closed    bool
}

// Config содержит параметры конфигурации движка.
type Config struct {
	DataDir         string
	WALDir          string
	MaxMemTableSize int64
}

// NewEngine создает новый экземпляр движка хранения, инициализирует подсистемы и восстанавливает состояние.
func NewEngine(cfg Config) (*Engine, error) {
	// Принудительное создание директорий для надежности
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("ошибка создания директории данных: %w", err)
	}
	if err := os.MkdirAll(cfg.WALDir, 0755); err != nil {
		return nil, fmt.Errorf("ошибка создания директории WAL: %w", err)
	}

	// 1. Инициализация WAL (Журнал предзаписи)
	// Используем размер сегмента по умолчанию 64 МБ, если не указано иное
	walConfig := wal.Config{
		Dir:         cfg.WALDir,
		SegmentSize: 64 * 1024 * 1024, 
	}
	walLog, err := wal.NewLog(walConfig)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания WAL: %w", err)
	}

	// 2. Инициализация MemTable (Таблица в памяти)
	mt := memtable.NewTable(cfg.MaxMemTableSize)

	engine := &Engine{
		memTable: mt,
		wal:      walLog,
		config:   cfg,
		sstables: make([]*sstable.Table, 0),
		closed:   false,
	}

	// 3. Восстановление состояния из WAL и SSTable при старте
	// Это критическая операция для обеспечения долговечности данных после сбоя
	if err := engine.recoverState(); err != nil {
		// При ошибке восстановления закрываем ресурсы перед выходом
		walLog.Close()
		return nil, fmt.Errorf("ошибка восстановления состояния: %w", err)
	}

	// 4. Запуск воркера компактификации
	// Воркер работает в фоновом режиме, объединяя файлы SSTable и удаляя старые данные
	engine.compactor = compaction.NewWorker(engine)
	go engine.compactor.Run()

	return engine, nil
}

// Put записывает ключ-значение в базу данных с гарантией ACID.
func (e *Engine) Put(key []byte, value []byte) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		return fmt.Errorf("двиок закрыт, запись невозможна")
	}

	// Шаг 1: Запись в WAL для обеспечения долговечности (Durability)
	if err := e.wal.Append(key, value); err != nil {
		return fmt.Errorf("ошибка записи в WAL: %w", err)
	}

	// Шаг 2: Запись в MemTable для скорости (Speed)
	inserted, err := e.memTable.Put(key, value)
	if err != nil {
		return fmt.Errorf("ошибка записи в MemTable: %w", err)
	}

	// Шаг 3: Проверка необходимости сброса (Flush) на диск
	// Если MemTable переполнена, мы блокируем запись и сбрасываем данные в SSTable
	if inserted && e.memTable.Size() >= e.config.MaxMemTableSize {
		if err := e.flushMemTable(); err != nil {
			return fmt.Errorf("ошибка сброса MemTable: %w", err)
		}
	}

	return nil
}

// Get retrieves значение по ключу, проверяя сначала память, затем диски.
func (e *Engine) Get(key []byte) ([]byte, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.closed {
		return nil, fmt.Errorf("движок закрыт, чтение невозможно")
	}

	// 1. Поиск в MemTable (самые свежие данные)
	val, found := e.memTable.Get(key)
	if found {
		// Возвращаем nil, если значение было удалено (tombstone)
		if val == nil {
			return nil, fmt.Errorf("ключ не найден (удален)")
		}
		return val, nil
	}

	// 2. Поиск в SSTables (от новых к старым)
	// Итерируемся в обратном порядке, так как новые файлы могут содержать более актуальные данные
	for i := len(e.sstables) - 1; i >= 0; i-- {
		val, found := e.sstables[i].Get(key)
		if found {
			if val == nil {
				return nil, fmt.Errorf("ключ не найден (удален)")
			}
			return val, nil
		}
	}

	return nil, fmt.Errorf("ключ не найден")
}

// Delete удаляет ключ из базы данных, устанавливая маркер удаления (tombstone).
func (e *Engine) Delete(key []byte) error {
	// Удаление реализуется как запись специального значения nil
	return e.Put(key, nil)
}

// Close корректно закрывает все ресурсы движка, останавливая фоновые процессы.
func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		return nil
	}
	e.closed = true

	// Остановка воркера компактификации
	if e.compactor != nil {
		e.compactor.Stop()
	}

	// Сброс остатков из MemTable на диск
	if err := e.flushMemTable(); err != nil {
		return fmt.Errorf("ошибка финального сброса MemTable: %w", err)
	}

	// Закрытие WAL
	if err := e.wal.Close(); err != nil {
		return fmt.Errorf("ошибка закрытия WAL: %w", err)
	}

	// Закрытие всех SSTable
	for _, sst := range e.sstables {
		if err := sst.Close(); err != nil {
			// Логируем ошибку, но продолжаем закрытие остальных
			fmt.Printf("Предупреждение: ошибка закрытия SSTable: %v\n", err)
		}
	}

	return nil
}

// flushMemTable сбрасывает содержимое памяти на диск в виде нового SSTable.
func (e *Engine) flushMemTable() error {
	// Если таблица пуста, сбрасывать нечего
	if e.memTable.Size() == 0 {
		return nil
	}

	// Сериализация MemTable в SSTable
	sst, err := sstable.CreateTableFromMemTable(e.memTable, e.config.DataDir)
	if err != nil {
		return fmt.Errorf("ошибка создания SSTable: %w", err)
	}

	// Добавление нового файла в список активных
	e.sstables = append(e.sstables, sst)

	// Очистка MemTable (создание новой)
	e.memTable = memtable.NewTable(e.config.MaxMemTableSize)

	// Усечение WAL после успешного сброса
	// В полной реализации здесь нужно передавать ID последнего записанного сегмента
	if err := e.wal.Truncate(); err != nil {
		return fmt.Errorf("ошибка усечения WAL: %w", err)
	}

	return nil
}

// recoverState восстанавливает данные при старте системы.
// Порядок: 1. Загрузка списка SSTable. 2. Проигрывание WAL.
func (e *Engine) recoverState() error {
	// 1. Сканирование директории данных для поиска существующих SSTable
	// (Реализация загрузки метаданных SSTable должна быть в пакете sstable)
	existingTables, err := sstable.LoadExistingTables(e.config.DataDir)
	if err != nil {
		return fmt.Errorf("ошибка загрузки списка SSTable: %w", err)
	}
	e.sstables = existingTables

	// 2. Воспроизведение записей из WAL
	// Читаем все записи с начала (или с последнего чекпоинта)
	entries, err := wal.Replay(e.config.WALDir, 0)
	if err != nil {
		return fmt.Errorf("ошибка чтения WAL при восстановлении: %w", err)
	}

	// Применяем записи к MemTable
	for _, entry := range entries {
		_, err := e.memTable.Put(entry.Key, entry.Value)
		if err != nil {
			return fmt.Errorf("ошибка применения записи WAL при восстановлении: %w", err)
		}
	}

	return nil
}