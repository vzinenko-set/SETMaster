package actioner

import "github.com/vzinenko-set/SETMaster/module-engine/internal/config"

// Actioner визначає інтерфейс для виконання дій та отримання їх назв.
type Actioner interface {
	Execute(ip string) error // Execute виконує дію для заданого IP.
	Name() string            // Name повертає назву дії.
}

// Діяч для роботи з Google Cloud Firewall
func NewGCPFirewall(cfg config.ActionerConfig) *GCPFirewall {
	return &GCPFirewall{cfg: cfg}
}

// Діяч для роботи з Google Cloud Storage.
func NewGCPStorage(cfg config.ActionerConfig) *GCPStorage {
	return &GCPStorage{cfg: cfg}
}

// Діяч для роботи sigma-форматом
func NewSigmaHQActioner(cfg config.ActionerConfig) *SigmaHQActioner {
	return &SigmaHQActioner{cfg: cfg}
}
