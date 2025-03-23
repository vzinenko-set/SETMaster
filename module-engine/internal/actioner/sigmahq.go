package actioner

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/storage"
	"github.com/vzinenko-set/SETMaster/module-engine/internal/config"
	"google.golang.org/api/option"
	"gopkg.in/yaml.v2"
)

// Структура для обробки подій у форматі SigmaHQ.
type SigmaHQActioner struct {
	cfg config.ActionerConfig // Конфігурація діяча
}

// Name повертає ім’я діяча.
func (s *SigmaHQActioner) Name() string {
	return "sigmahq" // Повертає статичне ім’я "sigmahq"
}

// Структура події у форматі SigmaHQ.
type SigmaHQEvent struct {
	Title       string   `yaml:"title"`       // Заголовок події
	ID          string   `yaml:"id"`          // Унікальний ідентифікатор події
	Description string   `yaml:"description"` // Опис події
	Level       string   `yaml:"level"`       // Рівень критичності події
	LogSource   struct { // Джерело логів
		Product string `yaml:"product"` // Продукт, що згенерував подію
		Service string `yaml:"service"` // Сервіс, що згенерував подію
	} `yaml:"logsource"`
	Detection struct { // Умови виявлення
		Selection map[string]string `yaml:"selection"` // Поля для фільтрації
		Condition string            `yaml:"condition"` // Умова спрацьовування
	} `yaml:"detection"`
	Fields         []string `yaml:"fields"`         // Поля, що включаються до події
	FalsePositives []string `yaml:"falsepositives"` // Можливі хибнопозитивні спрацьовування
}

// Конвертація події в SigmaHQ формат та запис в бакет Google Cloud Storage.
func (s *SigmaHQActioner) Execute(ip string) error {
	// Отримуємо назву бакета з конфігурації
	bucketName := s.cfg.BucketName
	if bucketName == "" {
		return fmt.Errorf("Відсутня назва бакету в конфігураційному файлі") // Помилка, якщо назва бакета відсутня
	}

	// Створюємо подію у форматі SigmaHQ
	sigmaEvent := SigmaHQEvent{
		Title:       fmt.Sprintf("Detected Failed SSH Login Attempt for IP %s", ip),        // Формуємо заголовок із IP
		ID:          fmt.Sprintf("event-%s-%s", ip, time.Now().Format("20060102T150405Z")), // Генеруємо унікальний ID
		Description: "Detects failed SSH login attempts based on Falco event",              // Задаємо опис
		Level:       "medium",                                                              // Встановлюємо середній рівень критичності
		LogSource: struct { // Налаштовуємо джерело логів
			Product string `yaml:"product"`
			Service string `yaml:"service"`
		}{
			Product: "falco", // Продукт - Falco
			Service: "ssh",   // Сервіс - SSH
		},
		Detection: struct { // Налаштовуємо умови виявлення
			Selection map[string]string `yaml:"selection"`
			Condition string            `yaml:"condition"`
		}{
			Selection: map[string]string{ // Фільтр для виявлення
				"rule":   "Detect Failed SSH Login Attempts", // Назва правила
				"fd.rip": ip,                                 // IP-адреса джерела
			},
			Condition: "selection", // Умова - відповідність фільтру
		},
		Fields:         []string{"rule", "fd.rip"},                // Поля для включення в подію
		FalsePositives: []string{"Legitimate SSH login attempts"}, // Можливі хибнопозитивні випадки
	}

	// Перетворюємо подію в YAML-формат
	yamlData, err := yaml.Marshal(&sigmaEvent)
	if err != nil {
		log.Printf("Невдало розподілили дані до YAML: %v", err) // Логуємо помилку парсингу
		return err
	}

	// Створюємо контекст для роботи з Google Cloud Storage
	ctx := context.Background()
	var client *storage.Client
	// Ініціалізуємо клієнт GCS залежно від наявності файлу credentials
	if s.cfg.CredentialsFile != "" {
		client, err = storage.NewClient(ctx, option.WithCredentialsFile(s.cfg.CredentialsFile)) // З файлом credentials
	} else {
		client, err = storage.NewClient(ctx) // Без файла, використовуємо Application Default Credentials (ADC)
	}
	if err != nil {
		log.Printf("Помилка при створенні клієнта GCS: %v", err) // Логуємо помилку створення клієнта
		return err
	}
	defer client.Close() // Закриваємо клієнт після завершення роботи

	// Генеруємо ім’я файлу з часовою міткою
	fileName := fmt.Sprintf("sigmahq/%s-%s.yaml", ip, time.Now().Format("20060102T150405Z"))
	bucket := client.Bucket(bucketName) // Отримуємо бакет
	obj := bucket.Object(fileName)      // Створюємо об’єкт для запису

	// Записуємо YAML-дані в бакет
	w := obj.NewWriter(ctx)
	if _, err := w.Write(yamlData); err != nil {
		log.Printf("Помилка при запису до GCS бакету %s: %v", bucketName, err) // Логуємо помилку запису
		return err
	}
	if err := w.Close(); err != nil {
		log.Printf("Помилка при записі до GCS: %v", err) // Логуємо помилку закриття записувача
		return err
	}

	// Логуємо успішний запис
	log.Printf("Подію в форматі SigmaHQ успішно записано на GCS бакет %s as %s", bucketName, fileName)
	return nil // Повертаємо nil у разі успіху
}
