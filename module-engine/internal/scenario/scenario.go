package scenario

import (
	"fmt"
	"log"
	"time"

	"github.com/vzinenko-set/SETMaster/module-engine/internal/actioner"
	"github.com/vzinenko-set/SETMaster/module-engine/internal/config"
	"github.com/vzinenko-set/SETMaster/module-engine/internal/db"
	"github.com/vzinenko-set/SETMaster/module-engine/internal/notifier"
	"github.com/vzinenko-set/SETMaster/module-engine/pkg/models"
)

// Cтруктура для управління сценаріями та подіями
type Manager struct {
	cfg           *config.Config               // Конфігурація системи
	actioners     map[string]actioner.Actioner // Мапа доступних actioners для виконання дій
	db            *db.SQLiteDB                 // Підключення до бази даних SQLite
	notifier      *notifier.SlackNotifier      // Система сповіщень через Slack
	cancel        map[string]chan struct{}     // Для скасування notifier timeout
	unblockCancel map[string]chan struct{}     // Для скасування unblock таймерів
}

// Створення нового менеджера сценаріїв
func NewManager(cfg *config.Config, actioners map[string]actioner.Actioner, db *db.SQLiteDB, notifier *notifier.SlackNotifier) *Manager {
	return &Manager{
		cfg:           cfg,                            // Ініціалізація конфігурації
		actioners:     actioners,                      // Ініціалізація actioners
		db:            db,                             // Ініціалізація бази даних
		notifier:      notifier,                       // Ініціалізація сповіщень
		cancel:        make(map[string]chan struct{}), // Ініціалізація мапи для скасування таймаутів сповіщень
		unblockCancel: make(map[string]chan struct{}), // Ініціалізація мапи для скасування таймерів розблокування
	}
}

// Обробка вхідної події
func (m *Manager) HandleEvent(scenarioName string, event models.Event) {
	log.Printf("Обробка події для IP %s, сценарій %s", event.IP, scenarioName)
	scenario, exists := m.cfg.Scenarios[scenarioName]
	if !exists {
		log.Printf("Сценарій %s не знайдено в конфігурації", scenarioName)
		return
	}
	// Перевірка відповідності правила події правилу сценарію
	if event.Rule != scenario.Rule {
		log.Printf("Правило події %s не відповідає правилу сценарію %s для IP %s", event.Rule, scenario.Rule, event.IP)
		return
	}
	// Отримання або створення запису про блокування для IP
	record, err := m.db.GetOrCreateBlockRecord(event.IP) // Отримуємо або створюємо запис у базі даних
	if err != nil {
		log.Printf("Помилка при роботі з базою даних для IP %s: %v", event.IP, err)
		return
	}
	currentTime := time.Now().Unix() // Поточний час у секундах з початку епохи Unix
	// Онулення лічильника спрацьовувань
	if record.BlockedAt == 0 && record.TriggerCount > 0 && (currentTime-record.LastEventTime) > int64(scenario.Params.TriggerWindow*60) {
		log.Printf("Онулення TriggerCount для IP %s", event.IP)
		record.TriggerCount = 0    // Скидаємо лічильник подій
		record.ActionTaken = false // Позначаємо, що дія ще не виконана
	}

	// Перевіряємо, чи сценарій уже активний або повідомлення вже відправлено
	if record.ActionTaken {
		log.Printf("Сценарій уже активовано для IP %s, пропускаємо виконання", event.IP)
		return
	}

	record.TriggerCount++              // Збільшуємо лічильник подій
	record.LastEventTime = currentTime // Оновлюємо час останньої події
	log.Printf("IP %s: TriggerCount = %d, необхідний = %d", event.IP, record.TriggerCount, scenario.Params.TriggerCount)

	if scenario.Params.TriggerCount <= 0 {
		log.Printf("Некоректне значення trigger_count %d для сценарію %s, встановлюємо за замовчуванням 1", scenario.Params.TriggerCount, scenarioName)
		scenario.Params.TriggerCount = 1 // Встановлюємо значення за замовчуванням, якщо параметр некоректний
	}

	// Викликаємо сценарій лише коли TriggerCount вперше досягає межі
	if record.TriggerCount == scenario.Params.TriggerCount {
		log.Printf("Trigger threshold reached for IP %s, executing scenario", event.IP)
		m.executeScenario(scenarioName, event.IP, record) // Виконуємо сценарій
	}

	if err := m.db.UpdateBlockRecord(record); err != nil {
		log.Printf("Не вдалося оновити запис у базі даних для IP %s: %v", event.IP, err)
	}
}

// Виконання сценарію
func (m *Manager) executeScenario(scenarioName, ip string, record *models.BlockRecord) {
	scenario := m.cfg.Scenarios[scenarioName] // Отримуємо конфігурацію сценарію
	log.Printf("Виконання сценарію %s для IP %s", scenarioName, ip)

	if scenario.Action.Notifier.Enabled && scenario.Action.Notifier.Name == "slack" {
		buttons := []notifier.SlackButton{} // Список кнопок для повідомлення в Slack
		for _, actName := range scenario.Action.Actioners {
			buttons = append(buttons, notifier.SlackButton{Name: actName, Value: actName}) // Додаємо кнопку для кожного actioner
		}
		buttons = append(buttons, notifier.SlackButton{Name: "Виконати всі дії", Value: "all"}) // Додаємо кнопку для виконання всіх дій

		message := fmt.Sprintf("Для ІР %s активовано сценарій %s", ip, scenarioName) // Формуємо текст повідомлення
		ts, err := m.notifier.SendMessageWithButtons(message, buttons)               // Відправляємо повідомлення з кнопками
		if err != nil {
			log.Printf("Не вдалося відправити повідомлення в Slack для IP %s: %v", ip, err)
		} else {
			log.Printf("Повідомлення в Slack успішно відправлено для IP %s з ts: %s", ip, ts)
		}

		cancelChan := make(chan struct{}) // Канал для скасування таймауту
		m.cancel[ip] = cancelChan         // Зберігаємо канал у мапі

		log.Printf("Встановлення таймауту сповіщення на %d хвилин для IP %s", scenario.Action.Notifier.Timeout, ip)
		time.AfterFunc(time.Duration(scenario.Action.Notifier.Timeout)*time.Minute, func() { // Запускаємо таймер
			select {
			case <-cancelChan:
				log.Printf("Таймаут сповіщення скасовано для IP %s", ip)
				return
			default:
				updatedRecord, _ := m.db.GetOrCreateBlockRecord(ip) // Оновлюємо запис для перевірки
				if !updatedRecord.ActionTaken {
					log.Printf("Жодної дії не обрано протягом таймауту для IP %s, виконуємо всі дії", ip)
					m.ExecuteAction("all", ip) // Виконуємо всі дії автоматично
					if err := m.notifier.UpdateMessage(ts, fmt.Sprintf("Автоматично виконано всі дії для IP %s", ip)); err != nil {
						log.Printf("Не вдалося оновити повідомлення в Slack для IP %s: %v", ip, err)
					} else {
						log.Printf("Повідомлення в Slack оновлено для IP %s після автоматичного виконання", ip)
					}
				} else {
					log.Printf("Дія вже виконана для IP %s протягом таймауту", ip)
				}
			}
			delete(m.cancel, ip) // Видаляємо канал із мапи після завершення
		})
	} else {
		log.Printf("Сповіщення відключено для сценарію %s, виконуємо всі actioners для IP %s", scenarioName, ip)
		m.ExecuteAction("all", ip) // Виконуємо всі дії, якщо сповіщення відключені
	}
}

// Виконання конкретної дії для IP-адреси
func (m *Manager) ExecuteAction(action, ip string) {
	scenario := m.cfg.Scenarios["block_ip"]      // Отримуємо сценарій блокування IP
	record, _ := m.db.GetOrCreateBlockRecord(ip) // Отримуємо запис для IP

	if record.BlockedAt > 0 && action != "all" {
		log.Printf("IP %s уже заблоковано, пропускаємо дію %s", ip, action)
		return
	}

	if action == "all" {
		for _, actName := range scenario.Action.Actioners {
			log.Printf("Виконання actioner %s для IP %s", actName, ip)
			if err := m.actioners[actName].Execute(ip); err != nil { // Виконуємо кожен actioner
				log.Printf("Не вдалося виконати actioner %s для IP %s: %v", actName, ip, err)
			}
		}
	} else if actioner, ok := m.actioners[action]; ok {
		log.Printf("Виконання actioner %s для IP %s", action, ip)
		if err := actioner.Execute(ip); err != nil { // Виконуємо конкретний actioner
			log.Printf("Не вдалося виконати actioner %s для IP %s: %v", action, ip, err)
			return
		}
	} else {
		log.Printf("Unknown дія %s для IP %s", action, ip)
		return
	}

	record.ActionTaken = true // Позначаємо, що дія виконана
	if action == "gcp_firewall" || action == "all" {
		baseUnblockAfter := int64(scenario.Params.UnblockAfter * 60)          // Базовий час у секундах
		multiplier := int64(record.BlockCount + 1)                            // Збільшуємо на основі кількості попередніх блокувань
		record.BlockedAt = time.Now().Unix()                                  // Час блокування
		record.UnblockAfter = time.Now().Unix() + baseUnblockAfter*multiplier // Час розблокування
		record.BlockCount++                                                   // Збільшуємо лічильник блокувань
		if cancelChan, ok := m.cancel[ip]; ok {
			close(cancelChan) // Закриваємо канал таймауту
			delete(m.cancel, ip)
			log.Printf("Скасовано таймаут сповіщення для IP %s через виконання дії", ip)
		}
		unblockCancelChan := make(chan struct{}) // Канал для скасування розблокування
		m.unblockCancel[ip] = unblockCancelChan  // Зберігаємо канал у мапі
		log.Printf("IP %s заблоковано, розблокування через %d секунд (BlockCount: %d)", ip, baseUnblockAfter*multiplier, record.BlockCount)
		go m.scheduleUnblock(ip, record, unblockCancelChan) // Запускаємо горутину для розблокування
	}
	if err := m.db.UpdateBlockRecord(record); err != nil {
		log.Printf("Не вдалося оновити запис блокування для IP %s: %v", ip, err)
	}
}

func (m *Manager) scheduleUnblock(ip string, record *models.BlockRecord, cancelChan chan struct{}) {
	select {
	case <-time.After(time.Until(time.Unix(record.UnblockAfter, 0))): // Чекаємо до часу розблокування
		log.Printf("Розблокування IP %s", ip)
		if firewall, ok := m.actioners["gcp_firewall"].(*actioner.GCPFirewall); ok {
			if err := firewall.Unblock(ip); err != nil { // Виконуємо розблокування через GCP Firewall
				log.Printf("Не вдалося розблокувати IP %s: %v", ip, err)
			}
		}
		record.BlockedAt = 0       // Скидаємо час блокування
		record.TriggerCount = 0    // Скидаємо лічильник подій
		record.ActionTaken = false // Позначаємо, що дія завершена
		if err := m.db.UpdateBlockRecord(record); err != nil {
			log.Printf("Не вдалося оновити запис блокування для IP %s після розблокування: %v", ip, err)
		}
	case <-cancelChan:
		log.Printf("Таймер розблокування скасовано для IP %s", ip)
	}
	delete(m.unblockCancel, ip) // Видаляємо канал із мапи після завершення
}

// Ручне розблокування через веб-сторінку
func (m *Manager) ManualUnblock(ip string) error {
	record, err := m.db.GetOrCreateBlockRecord(ip) // Отримуємо запис для IP
	if err != nil {
		return fmt.Errorf("Не вдалося отримати запис блокування для IP %s: %v", ip, err)
	}

	if record.BlockedAt == 0 {
		log.Printf("IP %s не заблоковано, дії не потрібні", ip)
		return nil
	}

	// Скасовуємо таймер notifier, якщо він є
	if cancelChan, ok := m.cancel[ip]; ok {
		close(cancelChan)
		delete(m.cancel, ip)
		log.Printf("Скасовано таймаут сповіщення для IP %s через ручне розблокування", ip)
	}

	// Скасовуємо таймер розблокування, якщо він є
	if unblockCancelChan, ok := m.unblockCancel[ip]; ok {
		close(unblockCancelChan)
		delete(m.unblockCancel, ip)
		log.Printf("Скасовано таймер розблокування для IP %s через ручне розблокування", ip)
	}

	// Виконуємо розблокування
	if firewall, ok := m.actioners["gcp_firewall"].(*actioner.GCPFirewall); ok {
		if err := firewall.Unblock(ip); err != nil {
			return fmt.Errorf("Не вдалося розблокувати IP %s: %v", ip, err)
		}
	}

	record.BlockedAt = 0       // Скидаємо час блокування
	record.TriggerCount = 0    // Скидаємо лічильник подій
	record.ActionTaken = false // Позначаємо, що дія завершена
	if err := m.db.UpdateBlockRecord(record); err != nil {
		return fmt.Errorf("Не вдалося оновити запис блокування для IP %s: %v", ip, err)
	}

	log.Printf("Успішно розблоковано IP %s вручну", ip)
	return nil
}
