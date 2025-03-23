package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

// Основна структура конфігурації програми
type Config struct {
	Server struct {
		Port          int               `yaml:"port"`           // Порт основного сервера
		DashboardPort int               `yaml:"dashboard_port"` // Порт для дашборду
		Aliases       map[string]string `yaml:"aliases"`        // Мапа псевдонімів для серверів
	} `yaml:"server"`
	Scenarios map[string]struct { // Налаштування сценаріїв
		Rule   string         `yaml:"rule"`   // Правило для спрацьовування сценарію
		Params ScenarioParams `yaml:"params"` // Параметри сценарію
		Action ScenarioAction `yaml:"action"` // Дія, що виконується при спрацьовуванні
	} `yaml:"scenarios"`
	Actioners map[string]ActionerConfig `yaml:"actioners"` // Налаштування виконавців дій
	Notifier  struct {                  // Налаштування системи сповіщень
		Slack struct {
			WebhookURL  string `yaml:"webhook_url"`  // URL вебхука для Slack
			CallbackURL string `yaml:"callback_url"` // URL для зворотних викликів
			BotToken    string `yaml:"bot_token"`    // Токен бота Slack
			Channel     string `yaml:"channel"`      // Канал для надсилання повідомлень
		} `yaml:"slack"`
	} `yaml:"notifier"`
}

// Параметри для сценаріїв
type ScenarioParams struct {
	TriggerCount  int `yaml:"trigger_count"`  // Кількість спрацьовувань для активації
	TriggerWindow int `yaml:"trigger_window"` // Часовий проміжок для підрахунку спрацьовувань (в секундах)
	UnblockAfter  int `yaml:"unblock_after"`  // Час після якого знімається блокування (в секундах)
}

// Дії які виконуються в сценарії
type ScenarioAction struct {
	Actioners []string       `yaml:"actioners"` // Список виконавців дій
	Notifier  NotifierConfig `yaml:"notifier"`  // Налаштування сповіщень для дії
}

// Налаштування нотифікатора
type NotifierConfig struct {
	Enabled bool   `yaml:"enabled"` // Увімкнено чи вимкнено сповіщення
	Name    string `yaml:"name"`    // Ім'я нотифікатора
	Timeout int    `yaml:"timeout"` // Таймаут для сповіщень (в секундах)
}

// Конфігурація діяча
type ActionerConfig struct {
	ProjectID       string `yaml:"project_id"`       // Ідентифікатор проєкту
	BucketName      string `yaml:"bucket_name"`      // Назва бакета для зберігання
	LogCount        int    `yaml:"log_count"`        // Кількість логів для обробки
	CredentialsFile string `yaml:"credentials_file"` // Шлях до файлу з обліковими даними
}

// Завантаження конфігурації з конфігураційного файлу
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path) // Читання файлу конфігурації
	if err != nil {
		return nil, err // Повернення помилки, якщо файл не вдалося прочитати
	}
	var cfg Config
	err = yaml.Unmarshal(data, &cfg) // Розпарсинг YAML у структуру Config
	return &cfg, err                 // Повернення конфігурації або помилки
}
