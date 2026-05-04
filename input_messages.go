package modelbind

const (
	MsgDecodeReadFailed        MessageID = "quvalid.decode.read_failed"
	MsgDecodeJSONFailed        MessageID = "quvalid.decode.json_failed"
	MsgDecodeInvalidModel      MessageID = "quvalid.decode.invalid_model"
	MsgDecodeUnsupportedField  MessageID = "quvalid.decode.unsupported_field"
	MsgDecodeInvalidFieldValue MessageID = "quvalid.decode.invalid_field_value"
)

func init() {
	DefaultMessageTemplates[MsgDecodeReadFailed] = "{{ .Func }}() failed: can not read request body: {{ .Error }}"
	DefaultMessageTemplates[MsgDecodeJSONFailed] = "{{ .Func }}() failed: invalid JSON: {{ .Error }}"
	DefaultMessageTemplates[MsgDecodeInvalidModel] = "{{ .Func }}() failed: model must be a struct or a pointer to struct"
	DefaultMessageTemplates[MsgDecodeUnsupportedField] = "{{ .Func }}() failed: field {{ .Field }} has unsupported type {{ .Type }}"
	DefaultMessageTemplates[MsgDecodeInvalidFieldValue] = "{{ .Func }}() failed: invalid value for field {{ .Field }}: {{ .Value }}"

	RussianMessageTemplates[MsgDecodeReadFailed] = "{{ .Func }}() ошибка: нельзя прочитать тело запроса: {{ .Error }}"
	RussianMessageTemplates[MsgDecodeJSONFailed] = "{{ .Func }}() ошибка: неверный JSON: {{ .Error }}"
	RussianMessageTemplates[MsgDecodeInvalidModel] = "{{ .Func }}() ошибка: модель должна быть структурой или указателем на структуру"
	RussianMessageTemplates[MsgDecodeUnsupportedField] = "{{ .Func }}() ошибка: поле {{ .Field }} имеет неподдерживаемый тип {{ .Type }}"
	RussianMessageTemplates[MsgDecodeInvalidFieldValue] = "{{ .Func }}() ошибка: неверное значение поля {{ .Field }}: {{ .Value }}"
}
