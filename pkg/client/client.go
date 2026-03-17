package client

import (
	"fmt"
	"net"
	"time"
)

// Client представляет активное соединение с сервером базы данных gosqld.
type Client struct {
	conn    net.Conn
	encoder *Encoder
	decoder *Decoder
	timeout time.Duration
}

// Config содержит параметры подключения.
type Config struct {
	Host    string
	Port    int
	Timeout time.Duration
}

// NewClient устанавливает соединение с сервером и возвращает экземпляр клиента.
func NewClient(cfg Config) (*Client, error) {
	address := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	
	conn, err := net.DialTimeout("tcp", address, cfg.Timeout)
	if err != nil {
		return nil, fmt.Errorf("не удалось установить соединение с сервером %s: %w", address, err)
	}

	client := &Client{
		conn:    conn,
		encoder: NewEncoder(conn),
		decoder: NewDecoder(conn),
		timeout: cfg.Timeout,
	}

	// Здесь можно добавить логику рукопожатия (Handshake) при необходимости
	// if err := client.handshake(); err != nil { ... }

	return client, nil
}

// Execute отправляет SQL-запрос на сервер и возвращает ответ.
func (c *Client) Execute(sql string) (*Response, error) {
	// Установка таймаута на запись
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, fmt.Errorf("ошибка установки таймаута записи: %w", err)
	}

	// Кодирование и отправка запроса
	if err := c.encoder.EncodeQuery(sql); err != nil {
		return nil, fmt.Errorf("ошибка кодирования запроса: %w", err)
	}

	// Установка таймаута на чтение
	if err := c.conn.SetReadDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, fmt.Errorf("ошибка установки таймаута чтения: %w", err)
	}

	// Чтение и декодирование ответа
	resp, err := c.decoder.DecodeResponse()
	if err != nil {
		return nil, fmt.Errorf("ошибка декодирования ответа: %w", err)
	}

	// Проверка статуса ответа
	if resp.Status == MsgTypeError {
		return nil, NewServerError(string(resp.Data))
	}

	if resp.Status != MsgTypeResult && resp.Status != MsgTypeQuery { // Query может быть эхом в некоторых реализациях
		// В зависимости от протокола, успешный ответ должен быть MsgTypeResult
		// Если сервер возвращает что-то иное, это аномалия
	}

	return resp, nil
}

// Close корректно закрывает сетевое соединение.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// IsConnected проверяет статус соединения.
func (c *Client) IsConnected() bool {
	return c.conn != nil
}