package db

import (
	"database/sql"

	"github.com/vzinenko-set/SETMaster/module-engine/pkg/models"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteDB struct {
	db *sql.DB
}

// Ініціалізація бази
func NewSQLiteDB(path string) (*SQLiteDB, error) {
	// Відкриваємо з'єднання з базою даних SQLite за вказаним шляхом
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	// Створення таблиці blocks, якщо вона ще не існує
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS blocks (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            ip TEXT UNIQUE,
            blocked_at INTEGER,
            unblock_after INTEGER,
            block_count INTEGER,
            trigger_count INTEGER,
            last_event_time INTEGER DEFAULT 0,
			action_taken BOOLEAN DEFAULT 0
        )
    `)
	// Повертаємо об'єкт SQLiteDB або помилку, якщо вона виникла
	return &SQLiteDB{db}, err
}

// Створюємо запис в таблиці
func (d *SQLiteDB) GetOrCreateBlockRecord(ip string) (*models.BlockRecord, error) {
	var record models.BlockRecord
	// Отримуємо запис із таблиці за IP-адресою
	err := d.db.QueryRow("SELECT id, ip, blocked_at, unblock_after, block_count, trigger_count, last_event_time, action_taken FROM blocks WHERE ip = ?", ip).Scan(
		&record.ID, &record.IP, &record.BlockedAt, &record.UnblockAfter, &record.BlockCount, &record.TriggerCount, &record.LastEventTime, &record.ActionTaken,
	)
	if err != nil && err != sql.ErrNoRows {
		// Якщо сталася помилка, крім відсутності запису, повертаємо її
		return nil, err
	}
	if err == sql.ErrNoRows {
		// Якщо запису немає, створюємо новий із початковими значеннями
		res, err := d.db.Exec("INSERT INTO blocks (ip, blocked_at, unblock_after, block_count, trigger_count, last_event_time, action_taken) VALUES (?, 0, 0, 0, 0, 0, 0)", ip)
		if err != nil {
			return nil, err
		}
		// Отримуємо ID нового запису
		id, _ := res.LastInsertId()
		// Ініціалізуємо запис із ID та IP
		record = models.BlockRecord{ID: int(id), IP: ip}
	}
	// Повертаємо вказівник на запис та nil як помилку
	return &record, nil
}

// Оновлюємо запис про блокування
func (d *SQLiteDB) UpdateBlockRecord(record *models.BlockRecord) error {
	// Оновлюємо всі поля запису в таблиці за IP-адресою
	_, err := d.db.Exec("UPDATE blocks SET blocked_at = ?, unblock_after = ?, block_count = ?, trigger_count = ?, last_event_time = ?, action_taken = ? WHERE ip = ?",
		record.BlockedAt, record.UnblockAfter, record.BlockCount, record.TriggerCount, record.LastEventTime, record.ActionTaken, record.IP)
	// Повертаємо результат виконання (помилку або nil)
	return err
}

// Перевіряємо чи вже було здійснено блокування
func (d *SQLiteDB) WasActionTaken(ip string) bool {
	var actionTaken bool
	// Отримуємо значення action_taken для вказаної IP-адреси
	d.db.QueryRow("SELECT action_taken FROM blocks WHERE ip = ?", ip).Scan(&actionTaken)
	// Повертаємо булеве значення, чи було виконано дію
	return actionTaken
}

// Вибірка всіх даних з бази
func (d *SQLiteDB) GetAllRecords() ([]models.BlockRecord, error) {
	// Виконуємо запит для отримання всіх записів із таблиці
	rows, err := d.db.Query("SELECT id, ip, blocked_at, unblock_after, block_count, trigger_count, last_event_time FROM blocks")
	if err != nil {
		return nil, err
	}
	// Закриваємо рядки після завершення роботи з ними
	defer rows.Close()

	var records []models.BlockRecord
	// Ітеруємося по всіх рядках результату запиту
	for rows.Next() {
		var r models.BlockRecord
		// Зчитуємо дані з кожного рядка в структуру BlockRecord
		if err := rows.Scan(&r.ID, &r.IP, &r.BlockedAt, &r.UnblockAfter, &r.BlockCount, &r.TriggerCount, &r.LastEventTime); err != nil {
			return nil, err
		}
		// Додаємо запис до слайсу
		records = append(records, r)
	}
	// Повертаємо слайс із усіма записами та nil як помилку
	return records, nil
}
