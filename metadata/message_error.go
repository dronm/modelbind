package metadata

type MessageError struct {
	ID   MessageID
	Data map[string]any
}

func NewMessageError(id MessageID, data map[string]any) error {
	if data == nil {
		data = make(map[string]any)
	}

	return &MessageError{
		ID:   id,
		Data: data,
	}
}

func (e *MessageError) Error() string {
	return Translate(LanguageEN, e.ID, e.Data)
}

func (e *MessageError) MessageID() MessageID {
	return e.ID
}

func (e *MessageError) MessageData() map[string]any {
	return e.Data
}
