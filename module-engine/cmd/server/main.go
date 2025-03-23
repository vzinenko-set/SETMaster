package main

import (
	"log"

	"github.com/vzinenko-set/SETMaster/module-engine/internal/config"
	"github.com/vzinenko-set/SETMaster/module-engine/internal/server"
)

func main() {
	// Завантаження конфігурації з файлу config.yaml
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		// Виведення повідомлення про помилку та завершення програми, якщо конфігурацію не вдалося завантажити
		log.Fatalf("Помилка при завнатаженні конфігураційного файлу: %v", err)
	}

	// Створення нового екземпляра сервера з використанням завантаженої конфігурації
	srv, err := server.NewServer(cfg)
	if err != nil {
		// Виведення повідомлення про помилку та завершення програми, якщо сервер не вдалося створити
		log.Fatalf("Помилка при створенні сервері: %v", err)
	}

	// Запуск сервера для обробки вхідних запитів
	srv.Start()
}
