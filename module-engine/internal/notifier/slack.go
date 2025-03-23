package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// Структуру для надсилання повідомлень у Slack
type SlackNotifier struct {
	WebhookURL  string // URL вебхука для надсилання повідомлень
	CallbackURL string // URL для обробки callback-запитів
	BotToken    string // Токен бота для автентифікації в Slack API
	Channel     string // Канал Slack, куди надсилатимуться повідомлення
}

// Структура для опису кнопки
type SlackButton struct {
	Name  string `json:"name"`  // Назва кнопки
	Value string `json:"value"` // Значення, яке повертається при натисканні
}

// Структура для опису дії
type SlackAction struct {
	Name  string `json:"name"`  // Унікальне ім'я дії
	Text  string `json:"text"`  // Текст, що відображається на кнопці
	Type  string `json:"type"`  // Тип дії (наприклад, "button")
	Value string `json:"value"` // Значення, асоційоване з дією
}

// Структура для опису вкладень повідомлення
type SlackAttachment struct {
	Text       string        `json:"text"`        // Текст вкладення
	CallbackID string        `json:"callback_id"` // Ідентифікатор для обробки callback
	Actions    []SlackAction `json:"actions"`     // Список дій (кнопок)
}

// Новий екземпляр SlackNotifier
func NewSlackNotifier(webhookURL, callbackURL, botToken, channel string) *SlackNotifier {
	return &SlackNotifier{
		WebhookURL:  webhookURL,
		CallbackURL: callbackURL,
		BotToken:    botToken,
		Channel:     channel,
	}
}

// Надсилання повідомлення з кнопками в Slack
func (s *SlackNotifier) SendMessageWithButtons(text string, buttons []SlackButton) (string, error) {
	log.Printf("Sending Slack message with %d buttons: %+v", len(buttons), buttons)
	// Перетворюємо кнопки у формат SlackAction
	var slackActions []SlackAction
	for _, button := range buttons {
		slackActions = append(slackActions, SlackAction{
			Name:  button.Name,
			Text:  button.Name,
			Type:  "button",
			Value: button.Value,
		})
	}
	// Створюємо структуру даних для відправки в Slack
	payload := struct {
		Channel     string            `json:"channel"`     // Канал для надсилання
		Text        string            `json:"text"`        // Основний текст повідомлення
		Attachments []SlackAttachment `json:"attachments"` // Вкладення з кнопками
	}{
		Channel: s.Channel,
		Text:    text,
		Attachments: []SlackAttachment{
			{
				Text:       "Оберіть дію:",    // Текст перед кнопками
				CallbackID: "block_ip_action", // Ідентифікатор callback
				Actions:    slackActions,      // Список кнопок
			},
		},
	}
	// Перетворюємо дані в JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Помилка при обробці повідомлення Slack: %v", err)
		return "", err
	}
	log.Printf("Дані для Slack: %s", string(jsonData))
	// Створюємо HTTP-запит до API Slack
	req, err := http.NewRequest("POST", "https://slack.com/api/chat.postMessage", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Помилка при створенні запиту до Slack: %v", err)
		return "", err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8") // Встановлюємо тип вмісту
	req.Header.Set("Authorization", "Bearer "+s.BotToken)             // Додаємо токен авторизації
	// Виконуємо запит
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Помилка при надсиланні повідомлення до Slack: %v", err)
		return "", err
	}
	defer resp.Body.Close() // Закриваємо тіло відповіді після завершення
	// Оброблюємо відповідь від Slack
	var result struct {
		Ok    bool   `json:"ok"`    // Успішність операції
		Ts    string `json:"ts"`    // Часова мітка повідомлення
		Error string `json:"error"` // Повідомлення про помилку, якщо є
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Помилка при декодуванні відповіді Slack: %v", err)
		return "", err
	}

	log.Printf("Відповідь Slack: ok=%v, ts=%s, error=%s", result.Ok, result.Ts, result.Error)
	if !result.Ok {
		log.Printf("Помилка API Slack: %s", result.Error)
		return "", fmt.Errorf("помилка API Slack: %s", result.Error)
	}

	log.Printf("Повідомлення Slack успішно надіслано з ts: %s", result.Ts)
	return result.Ts, nil // Повертаємо часову мітку повідомлення
}

// Оновлення існуючого повідомлення в Slack
func (s *SlackNotifier) UpdateMessage(ts, newText string) error {
	// Структура даних для оновлення повідомлення
	payload := struct {
		Channel     string            `json:"channel"`     // Канал, де знаходиться повідомлення
		Ts          string            `json:"ts"`          // Часова мітка повідомлення
		Text        string            `json Uran:"text"`   // Новий текст повідомлення
		Attachments []SlackAttachment `json:"attachments"` // Вказуємо порожній список вкладень
	}{
		Channel:     s.Channel,
		Ts:          ts,
		Text:        newText,
		Attachments: []SlackAttachment{}, // Очищаємо вкладення
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Помилка при виконанні оновлення Slack: %v", err)
		return fmt.Errorf("помилка при виконанні оновлення Slack: %v", err)
	}
	log.Printf("Дані для оновлення Slack: %s", string(jsonData))

	// Створюємо запит для оновлення повідомлення
	req, err := http.NewRequest("POST", "https://slack.com/api/chat.update", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Помилка при створенні запиту на оновлення Slack: %v", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8") // Встановлюємо тип вмісту
	req.Header.Set("Authorization", "Bearer "+s.BotToken)             // Додаємо токен авторизації

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Помилка при оновленні повідомлення Slack: %v", err)
		return err
	}
	defer resp.Body.Close() // Закриваємо тіло відповіді

	// Оброблюємо відповідь на оновлення
	var result struct {
		Ok    bool   `json:"ok"`    // Успішність операції
		Error string `json:"error"` // Повідомлення про помилку, якщо є
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Помилка при декодуванні відповіді на оновлення Slack: %v", err)
		return err
	}

	log.Printf("Відповідь на оновлення Slack: ok=%v, error=%s", result.Ok, result.Error)
	if !result.Ok {
		log.Printf("Помилка оновлення API Slack: %s", result.Error)
		return fmt.Errorf("помилка оновлення API Slack: %s", result.Error)
	}

	log.Printf("Повідомлення Slack успішно оновлено для ts: %s", ts)
	return nil // Повертаємо nil у разі успіху
}
