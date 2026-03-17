package client

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Encoder отвечает за сериализацию сообщений перед отправкой в сеть.
type Encoder struct {
	writer io.Writer
}

// NewEncoder создает новый экземпляр кодировщика.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{writer: w}
}

// EncodeQuery кодирует SQL-запрос в бинарный формат.
// Формат: [1 байт тип][4 байта длина][N байт данные]
func (e *Encoder) EncodeQuery(sql string) error {
	data := []byte(sql)
	length := uint32(len(data))

	// Запись типа сообщения
	if err := binary.Write(e.writer, binary.BigEndian, MsgTypeQuery); err != nil {
		return fmt.Errorf("ошибка записи типа сообщения: %w", err)
	}

	// Запись длины данных
	if err := binary.Write(e.writer, binary.BigEndian, length); err != nil {
		return fmt.Errorf("ошибка записи длины сообщения: %w", err)
	}

	// Запись самих данных
	if _, err := e.writer.Write(data); err != nil {
		return fmt.Errorf("ошибка записи тела сообщения: %w", err)
	}

	return nil
}

// Decoder отвечает за десериализацию входящих сообщений от сервера.
type Decoder struct {
	reader io.Reader
}

// NewDecoder создает новый экземпляр декодировщика.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{reader: r}
}

// DecodeResponse читает и разбирает ответ от сервера.
func (d *Decoder) DecodeResponse() (*Response, error) {
	// Чтение типа сообщения (1 байт)
	var msgType uint8
	if err := binary.Read(d.reader, binary.BigEndian, &msgType); err != nil {
		return nil, fmt.Errorf("ошибка чтения типа сообщения: %w", err)
	}

	// Чтение длины данных (4 байта)
	var length uint32
	if err := binary.Read(d.reader, binary.BigEndian, &length); err != nil {
		return nil, fmt.Errorf("ошибка чтения длины сообщения: %w", err)
	}

	// Чтение данных
	data := make([]byte, length)
	if _, err := io.ReadFull(d.reader, data); err != nil {
		return nil, fmt.Errorf("ошибка чтения тела сообщения: %w", err)
	}

	resp := &Response{
		Status: MessageType(msgType),
		Data:   data,
	}

	// Если это результат,可以尝试 прочитать количество затронутых строк (опционально, зависит от формата данных)
	// Для текущей версии считаем, что данные содержат всю необходимую информацию
	
	return resp, nil
}