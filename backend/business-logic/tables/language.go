package tables

import (
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// languageLogic is the business logic of table 8 Language
type languageLogic struct{}

func init() { gtables.RegisterLanguageLogic(languageLogic{}) }

func (languageLogic) OnInsert(rec *gtables.Language) error {
	return checkTranslationKey(rec)
}

func (languageLogic) OnModify(rec *gtables.Language) error {
	return checkTranslationKey(rec)
}

func (languageLogic) OnValidate_Translation_key(rec *gtables.Language) error {
	return checkTranslationKey(rec)
}

// checkTranslationKey checks the translation key format: blank or xx-XX (5 characters
// with a dash at position 3)
func checkTranslationKey(rec *gtables.Language) error {
	if rec.Translation_key.IsEmpty() {
		return nil
	}
	if key := rec.Translation_key.String(); len(key) != 5 || key[2] != '-' {
		return apperrors.FieldFormat(gtables.LanguageTableName, "translation_key", "xx-XX")
	}
	return nil
}
