package actioner

import (
	"context"
	"log"
	"strings"

	"github.com/vzinenko-set/SETMaster/module-engine/internal/config"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

// Структура для роботи з брандмауером Google Cloud Platform
type GCPFirewall struct {
	cfg config.ActionerConfig // Конфігурація для доступу до GCP
}

// Name повертає назву модуля
func (g *GCPFirewall) Name() string {
	return "gcp_firewall" // Унікальне ім'я для ідентифікації модуля
}

// Блокування IP у брандмауері GCP
func (g *GCPFirewall) Execute(ip string) error {
	ctx := context.Background() // Створюємо контекст для запитів до API
	// Ініціалізуємо сервіс Google Compute Engine з використанням файлу облікових даних
	svc, err := compute.NewService(ctx, option.WithCredentialsFile(g.cfg.CredentialsFile))
	if err != nil {
		return err // Повертаємо помилку, якщо не вдалося підключитися до сервісу
	}
	// Формуємо унікальне ім'я правила для брандмауера, замінюючи крапки в IP на дефіси
	ruleName := "block-" + strings.ReplaceAll(ip, ".", "-")
	ruleName = strings.ToLower(ruleName) // Переводимо ім'я в нижній регістр для консистентності

	// Створюємо нове правило брандмауера для блокування IP
	firewall := &compute.Firewall{
		Name:         ruleName,                                       // Ім'я правила
		SourceRanges: []string{ip + "/32"},                           // Діапазон IP для блокування (одинична адреса)
		Denied:       []*compute.FirewallDenied{{IPProtocol: "all"}}, // Блокуємо весь трафік
	}

	// Виконуємо запит на створення правила в GCP
	_, err = svc.Firewalls.Insert(g.cfg.ProjectID, firewall).Do()
	if err != nil {
		// Логуємо помилку, якщо не вдалося створити правило
		log.Printf("Не вдалося заблокувати IP %s: %v", ip, err)
		return err
	}
	// Логуємо успішне створення правила
	log.Printf("Успішно заблоковано IP %s з правилом %s", ip, ruleName)
	return nil // Повертаємо nil, якщо все пройшло успішно
}

// Видаляємо правило блокування для заданого IP
func (g *GCPFirewall) Unblock(ip string) error {
	ctx := context.Background() // Створюємо контекст для запитів до API
	// Ініціалізуємо сервіс Google Compute Engine з використанням файлу облікових даних
	svc, err := compute.NewService(ctx, option.WithCredentialsFile(g.cfg.CredentialsFile))
	if err != nil {
		return err // Повертаємо помилку, якщо не вдалося підключитися до сервісу
	}

	// Формуємо ім'я правила для видалення, аналогічно до створення
	ruleName := "block-" + strings.ReplaceAll(ip, ".", "-")
	ruleName = strings.ToLower(ruleName) // Переводимо ім'я в нижній регістр

	// Виконуємо запит на видалення правила з брандмауера
	if _, err := svc.Firewalls.Delete(g.cfg.ProjectID, ruleName).Do(); err != nil {
		// Логуємо помилку, якщо не вдалося видалити правило
		log.Printf("Не вдалося розблокувати IP %s: %v", ip, err)
		return err
	}
	// Логуємо успішне видалення правила
	log.Printf("Успішно розблоковано IP %s шляхом видалення правила %s", ip, ruleName)
	return nil // Повертаємо nil, якщо все пройшло успішно
}
