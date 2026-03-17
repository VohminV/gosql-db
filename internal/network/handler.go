package network

import (
	"gosql-db/internal/sql"
	"net"
)

// Handler обрабатывает входящие запросы клиента.
type Handler struct {
	conn      net.Conn
	sqlEngine *sql.Engine
}

func NewHandler(conn net.Conn, engine *sql.Engine) *Handler {
	return &Handler{conn: conn, sqlEngine: engine}
}

// Serve запускает цикл обработки запросов.
func (h *Handler) Serve() {
	// Чтение байтов, декодирование протокола, выполнение SQL, запись ответа
	// Детальная реализация бинарного протокола требуется для продакшена
}