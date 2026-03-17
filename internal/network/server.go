package network

import (
	"context"
	"fmt"
	"net"
	"gosql-db/internal/sql"
	"log"
)

// Server представляет TCP сервер базы данных.
type Server struct {
	host      string
	port      int
	sqlEngine *sql.Engine
	listener  net.Listener
}

// NewServer создает новый экземпляр сервера.
func NewServer(host string, port int, engine *sql.Engine) *Server {
	return &Server{
		host:      host,
		port:      port,
		sqlEngine: engine,
	}
}

// Start запускает прослушивание соединений.
func (s *Server) Start(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("ошибка привязки к порту: %w", err)
	}
	s.listener = lis

	defer s.listener.Close()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				log.Printf("Ошибка принятия соединения: %v", err)
				continue
			}
			go s.handleConnection(conn)
		}
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	// Инициализация обработчика протокола
	handler := NewHandler(conn, s.sqlEngine)
	handler.Serve()
}