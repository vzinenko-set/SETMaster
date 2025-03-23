package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/vzinenko-set/SETMaster/module-engine/internal/actioner"
	"github.com/vzinenko-set/SETMaster/module-engine/internal/config"
	"github.com/vzinenko-set/SETMaster/module-engine/internal/db"
	"github.com/vzinenko-set/SETMaster/module-engine/internal/notifier"
	"github.com/vzinenko-set/SETMaster/module-engine/internal/scenario"
	"github.com/vzinenko-set/SETMaster/module-engine/internal/web"
	"github.com/vzinenko-set/SETMaster/module-engine/pkg/models"
)

// Структура сервера з усіма необхідними компонентами
type Server struct {
	cfg       *config.Config               // Конфігурація сервера
	actioners map[string]actioner.Actioner // Мапа доступних діячів
	db        *db.SQLiteDB                 // Підключення до бази даних SQLite
	notifier  *notifier.SlackNotifier      // Система сповіщень через Slack
	scenarios *scenario.Manager            // Менеджер сценаріїв
}

// Створює новий екземпляр сервера з заданою конфігурацією
func NewServer(cfg *config.Config) (*Server, error) {
	// Ініціалізація бази даних SQLite
	db, err := db.NewSQLiteDB("blocks.db")
	if err != nil {
		return nil, err
	}

	// Ініціалізація діячів для різних сервісів
	actioners := map[string]actioner.Actioner{
		"gcp_firewall": actioner.NewGCPFirewall(cfg.Actioners["gcp_firewall"]), // Діяч для GCP Firewall
		"gcp_storage":  actioner.NewGCPStorage(cfg.Actioners["gcp_storage"]),   // Діяч для GCP Storage
		"sigmahq":      actioner.NewSigmaHQActioner(cfg.Actioners["sigmahq"]),  // Діяч для SigmaHQ
	}

	// Ініціалізація сповіщень через Slack
	slackNotifier := notifier.NewSlackNotifier(
		cfg.Notifier.Slack.WebhookURL,  // URL вебхука Slack
		cfg.Notifier.Slack.CallbackURL, // URL для зворотних викликів
		cfg.Notifier.Slack.BotToken,    // Токен бота Slack
		cfg.Notifier.Slack.Channel,     // Канал для сповіщень
	)

	// Ініціалізація менеджера сценаріїв
	scenarioMgr := scenario.NewManager(cfg, actioners, db, slackNotifier)

	// Повернення нового екземпляра сервера
	return &Server{cfg, actioners, db, slackNotifier, scenarioMgr}, nil
}

// Запуск серверу
func (s *Server) Start() {
	mux := http.NewServeMux() // Створення нового HTTP-мультиплексора

	// Реєстрація аліасів для обробки подій
	for alias, path := range s.cfg.Server.Aliases {
		log.Printf("Реєстрація аліасу %s за адресою %s", alias, path)
		mux.HandleFunc(path, s.handleEvent)
	}

	// Парсинг URL зворотного виклику для Slack
	callbackURL, err := url.Parse(s.cfg.Notifier.Slack.CallbackURL)
	if err != nil {
		log.Fatalf("Не вдалося розпарсити callback_url з конфігурації: %v", err)
	}

	// Визначення шляху для зворотного виклику
	callbackPath := callbackURL.Path
	if callbackPath == "" {
		callbackPath = "/callback" // Шлях за замовчуванням, якщо не вказано
	}
	log.Printf("Реєстрація зворотного виклику Slack за адресою %s", callbackPath)
	mux.HandleFunc(callbackPath, s.handleSlackCallback)

	// Запуск дашборду в окремій горутині
	go web.StartDashboard(s.cfg.Server.DashboardPort, s.db, s.scenarios)

	// Запуск HTTP-сервера
	log.Printf("Сервер запускається на порту :%d", s.cfg.Server.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", s.cfg.Server.Port), mux))
}

// Обробка вхідних події від Falco
func (s *Server) handleEvent(w http.ResponseWriter, r *http.Request) {
	// Структура для декодування події Falco
	var falcoEvent struct {
		Rule         string `json:"rule"`
		OutputFields struct {
			RemoteIP string `json:"fd.rip"`  // Віддалена IP-адреса
			Result   string `json:"evt.res"` // Результат події
			SourceIP string `json:"fd.sip"`  // Серверна IP-адреса
		} `json:"output_fields"`
		Time string `json:"time"` // Час події
	}

	// Декодування JSON-запиту в структуру falcoEvent
	if err := json.NewDecoder(r.Body).Decode(&falcoEvent); err != nil {
		log.Printf("Не вдалося декодувати подію: %v", err)
		http.Error(w, "Невірний JSON", http.StatusBadRequest)
		return
	}

	// Отримання віддаленої IP-адреси з події
	ip := falcoEvent.OutputFields.RemoteIP
	if ip == "" {
		log.Printf("Віддалена IP-адреса не знайдена в події")
		http.Error(w, "Відсутня IP-адреса", http.StatusBadRequest)
		return
	}

	// Логування отриманої події для відстеження
	log.Printf("Отримано подію для IP %s з правилом %s", ip, falcoEvent.Rule)

	// Створення структури події для подальшої обробки
	event := models.Event{
		IP:       ip,
		Rule:     falcoEvent.Rule,
		Result:   falcoEvent.OutputFields.Result,
		Time:     falcoEvent.Time,
		SourceIP: falcoEvent.OutputFields.SourceIP,
	}

	// Передача події в менеджер сценаріїв для обробки
	s.scenarios.HandleEvent("block_ip", event)
	w.WriteHeader(http.StatusOK) // Відправка успішної відповіді клієнту
}

// Обробка зворотніх викликів від Slack
func (s *Server) handleSlackCallback(w http.ResponseWriter, r *http.Request) {
	// Логування інформації про отриманий зворотний виклик
	log.Printf("Отримано зворотний виклик Slack: Метод=%s, URL=%s", r.Method, r.URL.String())

	// Перевірка, чи є метод запиту POST
	if r.Method != http.MethodPost {
		log.Printf("Невірний метод: %s", r.Method)
		http.Error(w, "Метод не дозволений", http.StatusMethodNotAllowed)
		return
	}

	// Парсинг даних форми з запиту
	if err := r.ParseForm(); err != nil {
		log.Printf("Не вдалося розпарсити форму: %v", err)
		http.Error(w, "Невірні дані форми", http.StatusBadRequest)
		return
	}

	// Отримання значення payload з форми
	payloadRaw := r.FormValue("payload")
	if payloadRaw == "" {
		log.Printf("Payload не знайдено у зворотному виклику")
		http.Error(w, "Відсутній payload", http.StatusBadRequest)
		return
	}
	log.Printf("Сирий payload зворотного виклику: %s", payloadRaw)

	// Структура для декодування payload з JSON
	var payload struct {
		CallbackID string `json:"callback_id"`
		Actions    []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"actions"`
		OriginalMessage struct {
			Text string `json:"text"`
		} `json:"original_message"`
	}

	// Декодування payload у визначену структуру
	if err := json.Unmarshal([]byte(payloadRaw), &payload); err != nil {
		log.Printf("Не вдалося декодувати payload: %v", err)
		http.Error(w, "Невірний форматJSON у payload", http.StatusBadRequest)
		return
	}

	// Логування розпарсеного зворотного виклику для дебагу
	log.Printf("Розпарсений зворотний виклик: CallbackID=%s, Actions=%+v, OriginalMessage=%s",
		payload.CallbackID, payload.Actions, payload.OriginalMessage.Text)

	// Обробка дії block_ip_action, якщо вона присутня
	if payload.CallbackID == "block_ip_action" && len(payload.Actions) > 0 {
		action := payload.Actions[0].Value
		var ip string

		// Витягування IP-адреси з тексту оригінального повідомлення
		if _, err := fmt.Sscanf(payload.OriginalMessage.Text, "IP %s triggered scenario block_ip", &ip); err != nil {
			log.Printf("Не вдалося витягти IP з повідомлення: %v", err)
			http.Error(w, "Не вдається розпарсити IP", http.StatusBadRequest)
			return
		}

		// Логування виконання дії для відстеження
		log.Printf("Виконання дії %s для IP %s із зворотного виклику Slack", action, ip)
		s.scenarios.ExecuteAction(action, ip)

		// Формування відповіді для Slack
		response := struct {
			Text        string `json:"text"`
			Attachments []struct {
				Text string `json:"text"`
			} `json:"attachments"`
			ReplaceOriginal bool `json:"replace_original"`
		}{
			Text: payload.OriginalMessage.Text,
			Attachments: []struct {
				Text string `json:"text"`
			}{
				{Text: fmt.Sprintf("Дію %s виконано для IP %s.", action, ip)},
			},
			ReplaceOriginal: true,
		}

		// Налаштування заголовка відповіді як JSON
		w.Header().Set("Content-Type", "application/json")
		// Відправка відповіді у форматі JSON
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Не вдалося сформувати відповідь: %v", err)
			return
		}
	} else {
		// Логування випадку, коли зворотний виклик не відповідає очікуванням
		log.Printf("Невірний зворотний виклик: CallbackID=%s, Кількість дій=%d", payload.CallbackID, len(payload.Actions))
		w.WriteHeader(http.StatusOK) // Повернення статусу OK для некоректного запиту
	}
}
