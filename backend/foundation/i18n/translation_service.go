package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// TranslationService manages multilanguage support
type TranslationService struct {
	translations map[string]map[string]string // [language][key]value
	defaultLang  string
	mu           sync.RWMutex
}

var (
	instance *TranslationService
	once     sync.Once
)

// GetInstance returns singleton translation service
func GetInstance() *TranslationService {
	once.Do(func() {
		instance = &TranslationService{
			translations: make(map[string]map[string]string),
			defaultLang:  "en-US",
		}
		// Load translations automatically on first access
		if err := instance.LoadTranslations(); err != nil {
			fmt.Printf("Warning: Failed to load translations: %v\n", err)
		}
	})
	return instance
}

// LoadTranslations loads all translation files at startup
func (s *TranslationService) LoadTranslations() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get project root (assuming we're in src/foundation/i18n)
	rootPath, err := findProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}

	translationsPath := filepath.Join(rootPath, "translations")

	// Check if translations directory exists
	if _, err := os.Stat(translationsPath); os.IsNotExist(err) {
		return fmt.Errorf("translations directory not found: %s", translationsPath)
	}

	// Find all language directories
	entries, err := os.ReadDir(translationsPath)
	if err != nil {
		return fmt.Errorf("failed to read translations directory: %w", err)
	}

	loadedCount := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		lang := entry.Name()
		langPath := filepath.Join(translationsPath, lang)

		// Initialize language map
		if s.translations[lang] == nil {
			s.translations[lang] = make(map[string]string)
		}

		// Load all YAML files in language directory
		langFiles, err := os.ReadDir(langPath)
		if err != nil {
			fmt.Printf("Warning: Failed to read language directory %s: %v\n", lang, err)
			continue
		}

		for _, file := range langFiles {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".yaml") {
				continue
			}

			filePath := filepath.Join(langPath, file.Name())
			if err := s.loadFile(lang, filePath); err != nil {
				fmt.Printf("Warning: Failed to load %s: %v\n", filePath, err)
			} else {
				loadedCount++
			}
		}
	}

	if loadedCount == 0 {
		return fmt.Errorf("no translation files loaded")
	}

	return nil
}

// loadFile loads a single translation YAML file
func (s *TranslationService) loadFile(lang, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var content map[string]interface{}
	if err := yaml.Unmarshal(data, &content); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Flatten nested structure into key-value pairs
	s.flattenMap("", content, lang)

	return nil
}

// flattenMap recursively flattens nested YAML into dot-notation keys
func (s *TranslationService) flattenMap(prefix string, m map[string]interface{}, lang string) {
	for key, value := range m {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]interface{}:
			// Recurse into nested maps
			s.flattenMap(fullKey, v, lang)
		case string:
			// Store string value
			s.translations[lang][fullKey] = v
		default:
			// Convert other types to string
			s.translations[lang][fullKey] = fmt.Sprint(v)
		}
	}
}

// Translate returns translated text for key in specified language
// Falls back to default language if not found
func (s *TranslationService) Translate(key, language string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Try requested language
	if langMap, ok := s.translations[language]; ok {
		if value, ok := langMap[key]; ok {
			return value
		}
	}

	// Fallback to default language
	if langMap, ok := s.translations[s.defaultLang]; ok {
		if value, ok := langMap[key]; ok {
			return value
		}
	}

	// Last resort: return key itself (makes missing translations visible)
	return key
}

// TableCaption returns table caption in specified language
func (s *TranslationService) TableCaption(tableName, language string) string {
	// Normalize table name: "Customer Ledger Entry" -> "customer_ledger_entry"
	normalizedName := normalizeTableName(tableName)
	key := fmt.Sprintf("tables.%s.caption", normalizedName)
	return s.Translate(key, language)
}

// FieldCaption returns field caption in specified language
func (s *TranslationService) FieldCaption(tableName, fieldName, language string) string {
	// Normalize table name: "Customer Ledger Entry" -> "customer_ledger_entry"
	normalizedName := normalizeTableName(tableName)
	key := fmt.Sprintf("tables.%s.fields.%s", normalizedName, fieldName)
	return s.Translate(key, language)
}

// OptionCaption returns option field value caption in specified language
func (s *TranslationService) OptionCaption(tableName, fieldName, optionValue, language string) string {
	// Normalize table name: "Customer Ledger Entry" -> "customer_ledger_entry"
	normalizedName := normalizeTableName(tableName)
	key := fmt.Sprintf("tables.%s.options.%s.%s", normalizedName, fieldName, optionValue)
	return s.Translate(key, language)
}

// PageCaption returns page caption in specified language
// pageName should be the normalized page name like "customer_card", "customer_list"
func (s *TranslationService) PageCaption(pageName, language string) string {
	// Normalize page name: "Customer Card" -> "customer_card"
	normalizedName := normalizeTableName(pageName)
	key := fmt.Sprintf("pages.%s.caption", normalizedName)
	return s.Translate(key, language)
}

// MenuItemCaption returns a translated menu item name.
// menuName is the English menu item name (e.g., "Job Queue").
// The lookup key is derived by lowercasing and replacing spaces with underscores.
// Falls back to the original name if no translation is found.
func (s *TranslationService) MenuItemCaption(menuName, language string) string {
	normalized := strings.ToLower(menuName)
	normalized = strings.ReplaceAll(normalized, " ", "_")
	key := fmt.Sprintf("common.menu.%s", normalized)
	translated := s.Translate(key, language)
	// If Translate returns the key itself, fall back to original name
	if translated == key {
		return menuName
	}
	return translated
}

// CommonTranslation returns a common translation by key
// key is the dot-notation path like "navigation.home", "actions.save"
func (s *TranslationService) CommonTranslation(key, language string) string {
	fullKey := fmt.Sprintf("common.%s", key)
	return s.Translate(fullKey, language)
}

// Message returns a message translation (success messages, dialogs, etc.)
// key is the message code like "MSG_LOGIN_SUCCESS", "DLG_DELETE_RECORD_TITLE"
func (s *TranslationService) Message(key, language string) string {
	fullKey := fmt.Sprintf("messages.%s", key)
	return s.Translate(fullKey, language)
}

// MessageWithParams returns a message translation with parameter substitution
// Replaces %1, %2, etc. with provided params
func (s *TranslationService) MessageWithParams(key, language string, params ...string) string {
	msg := s.Message(key, language)
	for i, param := range params {
		placeholder := fmt.Sprintf("%%%d", i+1)
		msg = strings.ReplaceAll(msg, placeholder, param)
	}
	return msg
}

// GetSupportedLanguages returns list of available languages
func (s *TranslationService) GetSupportedLanguages() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	langs := make([]string, 0, len(s.translations))
	for lang := range s.translations {
		langs = append(langs, lang)
	}
	return langs
}

// GetDefaultLanguage returns the default language
func (s *TranslationService) GetDefaultLanguage() string {
	return s.defaultLang
}

// GetAllMessages returns all message translations for a given language.
// Keys are returned without the "messages." prefix (e.g., "MSG_RECORD_CREATED").
// Falls back to default language for missing keys.
func (s *TranslationService) GetAllMessages(language string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]string)
	prefix := "messages."

	// First, populate with default language messages
	if langMap, ok := s.translations[s.defaultLang]; ok {
		for key, value := range langMap {
			if strings.HasPrefix(key, prefix) {
				shortKey := strings.TrimPrefix(key, prefix)
				result[shortKey] = value
			}
		}
	}

	// Then override with requested language (if different from default)
	if language != s.defaultLang {
		if langMap, ok := s.translations[language]; ok {
			for key, value := range langMap {
				if strings.HasPrefix(key, prefix) {
					shortKey := strings.TrimPrefix(key, prefix)
					result[shortKey] = value
				}
			}
		}
	}

	return result
}

// findProjectRoot finds the project root directory
func findProjectRoot() (string, error) {
	// Start from current working directory
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up until we find go.mod
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root without finding go.mod
			return "", fmt.Errorf("project root not found (no go.mod)")
		}
		dir = parent
	}
}

// normalizeTableName converts table names to snake_case for translation lookup
// Examples: "Customer" -> "customer", "Payment Terms" -> "payment_terms", "Customer Ledger Entry" -> "customer_ledger_entry"
func normalizeTableName(tableName string) string {
	// Convert to lowercase and replace spaces with underscores
	normalized := strings.ToLower(tableName)
	normalized = strings.ReplaceAll(normalized, " ", "_")
	return normalized
}
