package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"gosql-db/internal/network"
	"gosql-db/internal/storage"
	"gosql-db/internal/sql"
)

func main() {
	log.Println("Инициализация системы управления базами данных GosqlDB...")

	// Загрузка конфигурации
	configPath := os.Getenv("GOSQL_CONFIG")
	if configPath == "" {
		configPath = "config.yaml"
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Критическая ошибка загрузки конфигурации: %v", err)
	}

	// Инициализация подсистемы хранения
	engine, err := storage.NewEngine(cfg.Storage)
	if err != nil {
		log.Fatalf("Критическая ошибка инициализации движка хранения: %v", err)
	}
	defer func() {
		if closeErr := engine.Close(); closeErr != nil {
			log.Printf("Предупреждение при закрытии движка: %v", closeErr)
		}
	}()
	log.Println("Подсистема хранения инициализирована успешно.")

	// Инициализация SQL слоя
	sqlEngine := sql.NewEngine(engine)
	log.Println("SQL движок инициализирован.")

	// Инициализация сетевого сервера
	server := network.NewServer(cfg.Server.Host, cfg.Server.Port, sqlEngine)

	// Контекст для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запуск сервера в отдельной горутине
	go func() {
		log.Printf("Сервер слушает адрес %s:%d", cfg.Server.Host, cfg.Server.Port)
		if listenErr := server.Start(ctx); listenErr != nil {
			log.Fatalf("Ошибка запуска сетевого сервера: %v", listenErr)
		}
	}()

	// Обработка сигналов операционной системы
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	log.Printf("Получен сигнал завершения: %v. Начало безопасного отключения...", sig)

	cancel()
	log.Println("Система остановлена корректно.")
}