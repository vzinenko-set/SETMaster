package web

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/vzinenko-set/SETMaster/module-engine/internal/db"
	"github.com/vzinenko-set/SETMaster/module-engine/internal/scenario"
)

// Структура для відображення записів блокування зі статусом
type BlockRecordWithStatus struct {
	ID            int    // Ідентифікатор запису
	IP            string // IP-адреса
	Status        string // Поточний статус (Заблоковано/Не заблоковано)
	BlockedAt     string // Час початку блокування
	UnblockAfter  string // Час завершення блокування
	BlockCount    int    // Кількість блокувань
	TriggerCount  int    // Кількість спрацьовувань
	LastEventTime string // Час останньої події
}

// Структура для зберігання залежностей веб-дашборда
type Dashboard struct {
	db       *db.SQLiteDB      // Посилання на базу даних SQLite
	scenario *scenario.Manager // Посилання на менеджер сценаріїв
}

// Запуск веб-сторінки на вказаному порту
func StartDashboard(port int, db *db.SQLiteDB, mgr *scenario.Manager) {
	d := &Dashboard{db: db, scenario: mgr}                        // Ініціалізація Dashboard з переданими залежностями
	mux := http.NewServeMux()                                     // Створення нового HTTP-мультиплексора
	mux.HandleFunc("/", d.dashboardHandler)                       // Реєстрація обробника головної сторінки
	mux.HandleFunc("/unblock", d.unblockHandler)                  // Реєстрація обробника розблокування
	log.Printf("Dashboard starting on :%d", port)                 // Логування запуску дашборда
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), mux)) // Запуск HTTP-сервера
}

// Обробка HTTP-запитів для відображення дашборда
func (d *Dashboard) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	records, err := d.db.GetAllRecords() // Отримання всіх записів із бази даних
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError) // Помилка при збої бази даних
		return
	}

	currentTime := time.Now().Unix()              // Поточний час у форматі Unix timestamp
	var recordsWithStatus []BlockRecordWithStatus // Слайс для зберігання записів зі статусами
	for _, record := range records {
		status := "Not Blocked" // За замовчуванням IP не заблоковано
		if record.BlockedAt > 0 && record.UnblockAfter > currentTime {
			status = "Blocked" // Якщо час блокування активний, статус "Заблоковано"
		}

		blockedAt := "N/A" // Значення за замовчуванням для часу блокування
		if record.BlockedAt > 0 {
			blockedAt = time.Unix(record.BlockedAt, 0).Format("2006-01-02 15:04:05") // Форматування часу блокування
		}
		unblockAfter := "N/A" // Значення за замовчуванням для часу розблокування
		if record.UnblockAfter > 0 {
			unblockAfter = time.Unix(record.UnblockAfter, 0).Format("2006-01-02 15:04:05") // Форматування часу розблокування
		}
		lastEventTime := "N/A" // Значення за замовчуванням для часу останньої події
		if record.LastEventTime > 0 {
			lastEventTime = time.Unix(record.LastEventTime, 0).Format("2006-01-02 15:04:05") // Форматування часу останньої події
		}

		// Додавання запису зі статусом до слайсу
		recordsWithStatus = append(recordsWithStatus, BlockRecordWithStatus{
			ID:            record.ID,
			IP:            record.IP,
			Status:        status,
			BlockedAt:     blockedAt,
			UnblockAfter:  unblockAfter,
			BlockCount:    record.BlockCount,
			TriggerCount:  record.TriggerCount,
			LastEventTime: lastEventTime,
		})
	}

	// Парсинг HTML-шаблону для веб-сторінки
	tmpl, err := template.ParseFiles("internal/web/templates/dashboard.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError) // Помилка при збої шаблону
		return
	}
	tmpl.Execute(w, recordsWithStatus) // Виконання шаблону з переданими даними
}

// unblockHandler - обробник HTTP-запитів для ручного розблокування IP
func (d *Dashboard) unblockHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed) // Перевірка методу запиту
		return
	}

	ip := r.FormValue("ip") // Отримання IP-адреси з форми
	if ip == "" {
		http.Error(w, "IP not provided", http.StatusBadRequest) // Помилка, якщо IP не вказано
		return
	}

	log.Printf("Manual unblock requested for IP %s", ip) // Логування запиту на розблокування
	err := d.scenario.ManualUnblock(ip)                  // Виклик методу ручного розблокування
	if err != nil {
		log.Printf("Failed to manually unblock IP %s: %v", ip, err)           // Логування помилки
		http.Error(w, "Failed to unblock IP", http.StatusInternalServerError) // Помилка при збої розблокування
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther) // Перенаправлення на головну сторінку
}
