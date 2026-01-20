// Frontend i18n service for messages and UI translations
import { get } from 'svelte/store';
import { currentUser } from '$lib/stores/user';

// Message translations - bundled for now, could be loaded from API later
const messages: Record<string, Record<string, string>> = {
	'en-US': {
		// Success Messages - Records
		MSG_RECORD_CREATED: 'Record created',
		MSG_RECORD_UPDATED: 'Record updated',
		MSG_RECORD_DELETED: 'Record deleted',
		MSG_RECORD_SAVED: 'Record saved successfully',
		MSG_RECORD_DELETED_SUCCESS: 'Record deleted successfully',
		MSG_USER_CREATED: 'Initial user created successfully',

		// Welcome Messages
		MSG_WELCOME_BACK: 'Welcome back, %1',

		// Company Messages
		MSG_COMPANY_CREATED: 'Company "%1" created successfully',

		// Error Messages
		ERR_FAILED_DELETE: 'Failed to delete record',
		ERR_FAILED_OPEN_CARD: 'Failed to open card',
		ERR_FIELD_NOT_EXIST: '%1 \'%2\' does not exist',

		// Dialog Titles
		DLG_DELETE_RECORD_TITLE: 'Delete Record',
		DLG_CONFIRM_TITLE: 'Confirm',

		// Dialog Messages
		DLG_DELETE_RECORD_CONFIRM: 'Are you sure you want to delete this record?',
		DLG_DELETE_CONFIRM: 'Are you sure you want to delete this %1?',
		DLG_CONFIRM_DEFAULT: 'Are you sure?',

		// Button Labels
		BTN_DELETE: 'Delete',
		BTN_CANCEL: 'Cancel',
		BTN_CONFIRM: 'Confirm',
		BTN_OK: 'OK',
		BTN_YES: 'Yes',
		BTN_NO: 'No'
	},
	'nb-NO': {
		// Success Messages - Records
		MSG_RECORD_CREATED: 'Post opprettet',
		MSG_RECORD_UPDATED: 'Post oppdatert',
		MSG_RECORD_DELETED: 'Post slettet',
		MSG_RECORD_SAVED: 'Post lagret',
		MSG_RECORD_DELETED_SUCCESS: 'Post slettet',
		MSG_USER_CREATED: 'Første bruker opprettet',

		// Welcome Messages
		MSG_WELCOME_BACK: 'Velkommen tilbake, %1',

		// Company Messages
		MSG_COMPANY_CREATED: 'Firma "%1" opprettet',

		// Error Messages
		ERR_FAILED_DELETE: 'Kunne ikke slette post',
		ERR_FAILED_OPEN_CARD: 'Kunne ikke åpne kort',
		ERR_FIELD_NOT_EXIST: '%1 \'%2\' finnes ikke',

		// Dialog Titles
		DLG_DELETE_RECORD_TITLE: 'Slett post',
		DLG_CONFIRM_TITLE: 'Bekreft',

		// Dialog Messages
		DLG_DELETE_RECORD_CONFIRM: 'Er du sikker på at du vil slette denne posten?',
		DLG_DELETE_CONFIRM: 'Er du sikker på at du vil slette denne %1?',
		DLG_CONFIRM_DEFAULT: 'Er du sikker?',

		// Button Labels
		BTN_DELETE: 'Slett',
		BTN_CANCEL: 'Avbryt',
		BTN_CONFIRM: 'Bekreft',
		BTN_OK: 'OK',
		BTN_YES: 'Ja',
		BTN_NO: 'Nei'
	}
};

// Get the current user's language or default to en-US
function getCurrentLanguage(): string {
	const user = get(currentUser);
	return user?.language || 'en-US';
}

// Get a message by key in the current language
export function t(key: string, ...params: string[]): string {
	const lang = getCurrentLanguage();
	let message = messages[lang]?.[key] || messages['en-US']?.[key] || key;

	// Replace %1, %2, etc. with params
	params.forEach((param, index) => {
		message = message.replace(`%${index + 1}`, param);
	});

	return message;
}

// Get a message by key in a specific language
export function tLang(key: string, lang: string, ...params: string[]): string {
	let message = messages[lang]?.[key] || messages['en-US']?.[key] || key;

	// Replace %1, %2, etc. with params
	params.forEach((param, index) => {
		message = message.replace(`%${index + 1}`, param);
	});

	return message;
}

// Export message keys as constants for type safety
export const MSG = {
	RECORD_CREATED: 'MSG_RECORD_CREATED',
	RECORD_UPDATED: 'MSG_RECORD_UPDATED',
	RECORD_DELETED: 'MSG_RECORD_DELETED',
	RECORD_SAVED: 'MSG_RECORD_SAVED',
	RECORD_DELETED_SUCCESS: 'MSG_RECORD_DELETED_SUCCESS',
	USER_CREATED: 'MSG_USER_CREATED',
	WELCOME_BACK: 'MSG_WELCOME_BACK',
	COMPANY_CREATED: 'MSG_COMPANY_CREATED'
} as const;

export const ERR = {
	FAILED_DELETE: 'ERR_FAILED_DELETE',
	FAILED_OPEN_CARD: 'ERR_FAILED_OPEN_CARD',
	FIELD_NOT_EXIST: 'ERR_FIELD_NOT_EXIST'
} as const;

export const DLG = {
	DELETE_RECORD_TITLE: 'DLG_DELETE_RECORD_TITLE',
	DELETE_RECORD_CONFIRM: 'DLG_DELETE_RECORD_CONFIRM',
	DELETE_CONFIRM: 'DLG_DELETE_CONFIRM',
	CONFIRM_TITLE: 'DLG_CONFIRM_TITLE',
	CONFIRM_DEFAULT: 'DLG_CONFIRM_DEFAULT'
} as const;

export const BTN = {
	DELETE: 'BTN_DELETE',
	CANCEL: 'BTN_CANCEL',
	CONFIRM: 'BTN_CONFIRM',
	OK: 'BTN_OK',
	YES: 'BTN_YES',
	NO: 'BTN_NO'
} as const;
