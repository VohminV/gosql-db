.PHONY: build test clean bench

BINARY_NAME=gosqld
CMD_PATH=./cmd/gosqld

build:
	@echo "Начало компиляции бинарного файла для целевой платформы..."
	go build -o bin/$(BINARY_NAME) -ldflags="-s -w" $(CMD_PATH)
	@echo "Сборка завершена успешно."

test:
	@echo "Запуск полного набора тестов безопасности и целостности..."
	go test -race -cover ./...

clean:
	rm -rf bin/
	go clean

bench:
	@echo "Запуск бенчмарков производительности..."
	go test -bench=. -benchmem ./internal/storage/...

install:
	go install $(CMD_PATH)	