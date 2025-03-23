package main

import (
	"github.com/falcosecurity/plugin-sdk-go/pkg/sdk/plugins"
	"github.com/falcosecurity/plugin-sdk-go/pkg/sdk/plugins/extractor"
	"github.com/falcosecurity/plugin-sdk-go/pkg/sdk/plugins/source"
	hubble "github.com/vzinenko-set/SETMaster/hubble-plugin/pkg"
)

// Константи, що визначають основну інформацію про плагін
const (
	PluginID          uint32 = 6                                   // Унікальний ідентифікатор плагіна
	PluginName               = "hubble"                            // Назва плагіна
	PluginDescription        = "Hubble Events"                     // Опис плагіна
	PluginContact            = "github.com/falcosecurity/plugins/" // Контактна інформація
	PluginVersion            = "0.1.0"                             // Версія плагіна
	PluginEventSource        = "hubble"                            // Джерело подій плагіна
)

// Ініціалізація плагіна при завантаженні
func init() {
	// Встановлення фабрики для створення екземпляра плагіна
	plugins.SetFactory(func() plugins.Plugin {
		p := &hubble.Plugin{} // Створення нового екземпляра плагіна Hubble

		// Встановлення основної інформації про плагін
		p.SetInfo(
			PluginID,
			PluginName,
			PluginDescription,
			PluginContact,
			PluginVersion,
			PluginEventSource,
		)

		extractor.Register(p) // Реєстрація плагіна як екстрактора
		source.Register(p)    // Реєстрація плагіна як джерела подій

		return p // Повернення ініціалізованого плагіна
	})
}

// Точка входу програми (порожня, оскільки це плагін)
func main() {}
