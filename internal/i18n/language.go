package i18n

import (
	"sync"
)

// Language represents a supported language
type Language string

const (
	LangChinese Language = "zh"
	LangEnglish Language = "en"
)

var (
	currentLang Language = LangChinese
	mu          sync.RWMutex
)

// SetLanguage sets the current language
func SetLanguage(lang Language) {
	mu.Lock()
	defer mu.Unlock()
	currentLang = lang
}

// GetLanguage returns the current language
func GetLanguage() Language {
	mu.RLock()
	defer mu.RUnlock()
	return currentLang
}

// T returns the translated string for the given key in the current language.
// If the key is not found, it returns the key itself.
func T(key string) string {
	mu.RLock()
	lang := currentLang
	mu.RUnlock()

	switch lang {
	case LangEnglish:
		if v, ok := enDict[key]; ok {
			return v
		}
	default:
		if v, ok := zhDict[key]; ok {
			return v
		}
	}
	return key
}

// Tlang returns the translated string for the given key in the specified language.
func Tlang(key string, lang Language) string {
	switch lang {
	case LangEnglish:
		if v, ok := enDict[key]; ok {
			return v
		}
	default:
		if v, ok := zhDict[key]; ok {
			return v
		}
	}
	return key
}
