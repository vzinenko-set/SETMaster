package models

// Структура отриманої інформаційної події
type Event struct {
	IP       string `json:"ip"`
	Rule     string
	Result   string
	Time     string
	SourceIP string
}

// Структура запису в базу
type BlockRecord struct {
	ID            int    //  Ідентифікатор запису
	IP            string // IP-адреса з інформаційного потоку
	BlockedAt     int64  // Час початку блокування ІР
	UnblockAfter  int64  // Час розблокування ІР
	BlockCount    int    // Лічильник циклів блокуань
	TriggerCount  int    // Кількість подій повязаних з ІР
	LastEventTime int64  // Час останньої події
	ActionTaken   bool   // Інформація про блокування
}
