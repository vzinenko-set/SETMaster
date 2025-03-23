package hubble

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alecthomas/jsonschema"

	"github.com/cilium/cilium/api/v1/observer"
	"github.com/falcosecurity/plugin-sdk-go/pkg/sdk"
	"github.com/falcosecurity/plugin-sdk-go/pkg/sdk/plugins"
	"github.com/falcosecurity/plugin-sdk-go/pkg/sdk/plugins/source"
	"google.golang.org/grpc"
)

var (
	ID          uint32 // Унікальний ідентифікатор плагіну
	Name        string // Назва плагіну
	Description string // Опис плагіну
	Contact     string // Контактна інформація
	Version     string // Версія плагіну
	EventSource string // Джерело подій
)

// PluginConfig містить конфігурацію плагіну
type PluginConfig struct {
	HubbleAddress string `json:"hubbleAddress" jsonschema:"description=Address of Hubble API (Default: localhost:4245)"` // Адреса Hubble API
}

// Plugin представляє основний плагін Hubble
type Plugin struct {
	plugins.BasePlugin              // Базова структура плагіну від SDK
	Config             PluginConfig // Конфігурація плагіну
}

// SetDefault встановлює значення за замовчуванням для конфігурації
func (p *PluginConfig) setDefault() {
	p.HubbleAddress = "localhost:4245" // Встановлює адресу за замовчуванням для Hubble API
}

// SetInfo встановлює інформацію про плагін
func (p *Plugin) SetInfo(id uint32, name, description, contact, version, eventSource string) {
	ID = id                   // Зберігає ідентифікатор плагіну
	Name = name               // Зберігає назву плагіну
	Contact = contact         // Зберігає контактну інформацію
	Version = version         // Зберігає версію плагіну
	EventSource = eventSource // Зберігає джерело подій
}

// Info повертає інформацію про плагін
func (p *Plugin) Info() *plugins.Info {
	return &plugins.Info{ // Повертає структуру з інформацією про плагін
		ID:          ID,          // Ідентифікатор плагіну
		Name:        Name,        // Назва плагіну
		Description: Description, // Опис плагіну
		Contact:     Contact,     // Контактна інформація
		Version:     Version,     // Версія плагіну
		EventSource: EventSource, // Джерело подій
	}
}

// InitSchema повертає JSON-схему для конфігурації плагіну
func (p *Plugin) InitSchema() *sdk.SchemaInfo {
	reflector := jsonschema.Reflector{ // Ініціалізує рефлектор для створення JSON-схеми
		RequiredFromJSONSchemaTags: true, // Усі властивості необов’язкові за замовчуванням
		AllowAdditionalProperties:  true, // Дозволяє нерозпізнані властивості без помилок парсингу
	}
	if schema, err := reflector.Reflect(&PluginConfig{}).MarshalJSON(); err == nil { // Генерує та серіалізує схему
		return &sdk.SchemaInfo{ // Повертає схему у вигляді структури SDK
			Schema: string(schema), // JSON-схема у вигляді рядка
		}
	}
	return nil // Повертає nil у разі помилки
}

// Init ініціалізує плагін з конфігурацією
func (p *Plugin) Init(config string) error {
	p.Config.setDefault()                            // Встановлює значення конфігурації за замовчуванням
	return json.Unmarshal([]byte(config), &p.Config) // Розпарсує JSON-конфігурацію у структуру
}

// Список полів, які можна витягувати з подій Hubble
func (p *Plugin) Fields() []sdk.FieldEntry {
	return []sdk.FieldEntry{ // Повертає масив полів, доступних для витягування
		{Type: "string", Name: "hubble.event_type", Desc: "Type of the event"},        // Тип події
		{Type: "string", Name: "hubble.source_ip", Desc: "Source ip"},                 // Джерельна IP-адреса
		{Type: "string", Name: "hubble.destination_ip", Desc: "Destination ip"},       // Цільова IP-адреса
		{Type: "string", Name: "hubble.traffic_direction", Desc: "traffic_direction"}, // Напрямок трафіку
		{Type: "string", Name: "hubble.flow_type", Desc: "flow type"},                 // Тип потоку
		{Type: "string", Name: "hubble.pod_name", Desc: "pod name"},                   // Назва pod’у
		{Type: "string", Name: "hubble.verdict", Desc: "Verdict of the event"},        // Вердикт події
		{Type: "string", Name: "hubble.summary", Desc: "Summary of the event"},        // Підсумок події
	}
}

// Extract витягує значення поля з події Hubble
func (p *Plugin) Extract(req sdk.ExtractRequest, evt sdk.EventReader) error {
	var flow observer.GetFlowsResponse                                  // Оголошує змінну для зберігання події
	if err := json.NewDecoder(evt.Reader()).Decode(&flow); err != nil { // Декодує подію з JSON
		return err // Повертає помилку, якщо декодування не вдалося
	}

	switch req.Field() { // Вибирає поле для витягування
	case "hubble.event_type":
		req.SetValue(flow.GetFlow().GetEventType().String()) // Встановлює тип події
	case "hubble.source_ip":
		req.SetValue(flow.GetFlow().GetIP().GetSource()) // Встановлює джерельну IP
	case "hubble.destination_ip":
		req.SetValue(flow.GetFlow().GetIP().GetDestination()) // Встановлює цільову IP
	case "hubble.traffic_direction":
		req.SetValue(flow.GetFlow().GetTrafficDirection().String()) // Встановлює напрямок трафіку
	case "hubble.flow_type":
		req.SetValue(flow.GetFlow().GetType().String()) // Встановлює тип потоку
	case "hubble.pod_name":
		req.SetValue(flow.GetFlow().GetDestination().GetPodName()) // Встановлює назву pod’у
	case "hubble.verdict":
		req.SetValue(flow.GetFlow().GetVerdict().String()) // Встановлює вердикт
	case "hubble.summary":
		req.SetValue(flow.GetFlow().GetSummary()) // Встановлює підсумок
	default:
		return fmt.Errorf("no known field: %s", req.Field()) // Повертає помилку для невідомого поля
	}

	return nil // Повертає nil у разі успіху
}

// Встановлення з'єднання з Hubble
func (p *Plugin) Open(params string) (source.Instance, error) {
	conn, err := grpc.Dial(p.Config.HubbleAddress, grpc.WithInsecure()) // Встановлює gRPC-з’єднання з Hubble
	if err != nil {                                                     // Перевіряє помилку підключення
		return nil, fmt.Errorf("failed to connect to Hubble: %v", err) // Повертає помилку, якщо з’єднання не вдалося
	}

	client := observer.NewObserverClient(conn)                    // Створює клієнт для Hubble API
	request := &observer.GetFlowsRequest{Follow: true}            // Формує запит на отримання потоку подій
	stream, err := client.GetFlows(context.Background(), request) // Отримує потік подій від Hubble
	if err != nil {                                               // Перевіряє помилку отримання потоку
		return nil, fmt.Errorf("failed to get flows: %v", err) // Повертає помилку, якщо потік не отримано
	}

	eventC := make(chan source.PushEvent) // Створює канал для передачі подій
	go func() {                           // Запускає горутину для обробки потоку
		defer close(eventC) // Закриває канал після завершення
		for {               // Нескінченний цикл для отримання подій
			flow, err := stream.Recv() // Отримує наступну подію з потоку
			if err != nil {            // Перевіряє помилку отримання
				eventC <- source.PushEvent{Err: err} // Надсилає помилку в канал
				return                               // Завершує горутину
			}
			bytes, err := json.Marshal(flow) // Серіалізує подію в JSON
			if err != nil {                  // Перевіряє помилку серіалізації
				eventC <- source.PushEvent{Err: err} // Надсилає помилку в канал
				return                               // Завершує горутину
			}
			eventC <- source.PushEvent{Data: bytes} // Надсилає серіалізовану подію в канал
		}
	}()

	instance, err := source.NewPushInstance(eventC) // Створює екземпляр для передачі подій
	if err != nil {                                 // Перевіряє помилку створення екземпляра
		return nil, err // Повертає помилку
	}

	return instance, nil // Повертає екземпляр потоку подій
}
