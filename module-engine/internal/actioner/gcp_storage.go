package actioner

import (
	"context"
	"fmt"

	"github.com/vzinenko-set/SETMaster/module-engine/internal/config"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

// Структура для роботи зі сховищем Google Cloud Platform
type GCPStorage struct {
	cfg config.ActionerConfig // Конфігурація для доступу до сховища
}

// Name - метод повертає назву сервісу
func (g *GCPStorage) Name() string {
	return "gcp_storage" // Повертає ідентифікатор сервісу
}

// Метод для виконання запису логів у Google Cloud Storage
func (g *GCPStorage) Execute(ip string) error {
	ctx := context.Background() // Створення контексту для операцій з API

	// Ініціалізація клієнта Google Cloud Storage із файлом облікових даних
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(g.cfg.CredentialsFile))
	if err != nil {
		return err // Повернення помилки, якщо клієнт не вдалося створити
	}

	// Вибір бакета (сховища) з конфігурації
	bucket := client.Bucket(g.cfg.BucketName)

	// Формуємо унікальне ім’я файлу на основі IP-адреси
	obj := bucket.Object(fmt.Sprintf("logs-%s.txt", ip))

	// Створення об’єкта для запису даних у сховище
	w := obj.NewWriter(ctx)

	// Запис текстових даних у файл у сховищі
	if _, err := w.Write([]byte("Успішно записано дані на сховище " + ip)); err != nil {
		return err // Повернення помилки, якщо запис не вдався
	}

	// Завершення запису та закриття з’єднання
	return w.Close()
}
