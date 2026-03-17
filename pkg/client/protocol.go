package client

// MessageType определяет тип сообщения в бинарном протоколе.
type MessageType uint8

const (
	MsgTypeQuery   MessageType = 0x01 // Запрос SQL
	MsgTypeResult  MessageType = 0x02 // Успешный результат
	MsgTypeError   MessageType = 0x03 // Ошибка выполнения
	MsgTypeAuth    MessageType = 0x04 // Аутентификация (резерв)
	MsgTypeHandshake MessageType = 0x05 // Рукопожатие при подключении
)

// QueryRequest представляет структуру запроса от клиента к серверу.
type QueryRequest struct {
	SQL string
}

// Response представляет структуру ответа от сервера клиенту.
type Response struct {
	Status  MessageType
	Data    []byte  // Полезная нагрузка (результат запроса или сообщение об ошибке)
	RowsAffected int64
}