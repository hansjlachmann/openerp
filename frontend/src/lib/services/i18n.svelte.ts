// Frontend i18n service - loads translations from backend API
// Uses $state for reactivity so components re-render when translations load

// Reactive store holding all loaded translations (flat key-value map)
let translations = $state<Record<string, string>>({});

// Track whether translations have been loaded
let loaded = $state(false);

/**
 * Load translations from the backend API.
 * Reads the user's translation_key from localStorage to request the correct language.
 * Called on app init and after language change.
 */
export async function loadTranslations(): Promise<void> {
	try {
		// Read user's translation_key from localStorage for language preference
		let lang = '';
		try {
			const raw = localStorage.getItem('currentUser');
			if (raw) {
				const user = JSON.parse(raw);
				lang = user.translation_key || '';
			}
		} catch {
			// Ignore localStorage parse errors
		}

		const url = lang
			? `/api/translations?lang=${encodeURIComponent(lang)}`
			: '/api/translations';
		const response = await fetch(url);
		if (!response.ok) return;
		const result = await response.json();
		if (result.success && result.data) {
			translations = result.data;
			loaded = true;
		}
	} catch {
		// Silently fail - translations will fall back to keys
	}
}

/**
 * Check if translations have been loaded from the API.
 */
export function translationsLoaded(): boolean {
	return loaded;
}

/**
 * Get a message by key in the current language (loaded from API).
 * Falls back to the key itself if not found.
 * Reactive: components calling t() re-render when translations load.
 */
export function t(key: string, ...params: string[]): string {
	let message = translations[key] || key;

	// Replace %1, %2, etc. with params
	params.forEach((param, index) => {
		message = message.replace(`%${index + 1}`, param);
	});

	return message;
}

/**
 * Get a message by key. Same as t() since translations are loaded per-language from the API.
 * The lang parameter is kept for API compatibility but ignored -
 * the backend already returns translations for the session's language.
 */
export function tLang(key: string, _lang: string, ...params: string[]): string {
	return t(key, ...params);
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
	COMPANY_CREATED: 'MSG_COMPANY_CREATED',
	LOADING: 'MSG_LOADING',
	PROCESSING: 'MSG_PROCESSING',
	SAVING: 'MSG_SAVING',
	SAVED: 'MSG_SAVED',
	NO_MATCHES: 'MSG_NO_MATCHES',
	NO_RECORDS: 'MSG_NO_RECORDS',
	LOADING_MENU: 'MSG_LOADING_MENU',
	NO_MENU_ITEMS: 'MSG_NO_MENU_ITEMS',
	CODEUNIT_SUCCESS: 'MSG_CODEUNIT_SUCCESS',
	JOB_COMPLETED: 'MSG_JOB_COMPLETED'
} as const;

export const ERR = {
	FAILED_DELETE: 'ERR_FAILED_DELETE',
	FAILED_OPEN_CARD: 'ERR_FAILED_OPEN_CARD',
	FIELD_NOT_EXIST: 'ERR_FIELD_NOT_EXIST',
	NO_RECORD_SELECTED: 'ERR_NO_RECORD_SELECTED',
	UNKNOWN_OBJECT_TYPE: 'ERR_UNKNOWN_OBJECT_TYPE',
	INVALID_CODEUNIT_ID: 'ERR_INVALID_CODEUNIT_ID',
	FAILED_SAVE_RECORD: 'ERR_FAILED_SAVE_RECORD',
	FAILED_LOAD_CARD_PAGE: 'ERR_FAILED_LOAD_CARD_PAGE',
	FAILED_START_JOB: 'ERR_FAILED_START_JOB',
	CODEUNIT_FAILED: 'ERR_CODEUNIT_FAILED',
	FAILED_LOAD_MENU: 'ERR_FAILED_LOAD_MENU',
	SAVE_BLOCKED: 'ERR_SAVE_BLOCKED',
	VALIDATION_FAILED: 'ERR_VALIDATION_FAILED',
	CONNECTION_LOST: 'ERR_CONNECTION_LOST',
	LOGIN_MISSING_CREDENTIALS: 'ERR_LOGIN_MISSING_CREDENTIALS',
	LOGIN_NO_COMPANY: 'ERR_LOGIN_NO_COMPANY',
	LOGIN_FAILED: 'ERR_LOGIN_FAILED',
	INVALID_CREDENTIALS: 'ERR_INVALID_CREDENTIALS',
	GENERIC_RETRY: 'ERR_GENERIC_RETRY',
	SETUP_MISSING_FIELDS: 'ERR_SETUP_MISSING_FIELDS',
	PASSWORD_MISMATCH: 'ERR_PASSWORD_MISMATCH',
	PASSWORD_TOO_SHORT: 'ERR_PASSWORD_TOO_SHORT',
	FAILED_CREATE_USER: 'ERR_FAILED_CREATE_USER',
	SETUP_ERROR: 'ERR_SETUP_ERROR',
	COMPANY_NAME_REQUIRED: 'ERR_COMPANY_NAME_REQUIRED',
	FAILED_CREATE_COMPANY: 'ERR_FAILED_CREATE_COMPANY',
	GENERIC: 'ERR_GENERIC'
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

export const LOGIN = {
	SIGN_IN: 'LOGIN_SIGN_IN',
	INITIAL_SETUP: 'LOGIN_INITIAL_SETUP',
	SIGNING_IN: 'LOGIN_SIGNING_IN',
	CREATING: 'LOGIN_CREATING',
	USER_ID: 'LOGIN_USER_ID',
	FULL_NAME: 'LOGIN_FULL_NAME',
	EMAIL: 'LOGIN_EMAIL',
	PASSWORD: 'LOGIN_PASSWORD',
	CONFIRM_PASSWORD: 'LOGIN_CONFIRM_PASSWORD',
	COMPANY: 'LOGIN_COMPANY',
	COMPANY_NAME: 'LOGIN_COMPANY_NAME',
	CREATE_INITIAL_USER: 'LOGIN_CREATE_INITIAL_USER',
	CREATE_COMPANY: 'LOGIN_CREATE_COMPANY',
	BACK_TO_LOGIN: 'LOGIN_BACK_TO_LOGIN',
	SIGN_OUT: 'LOGIN_SIGN_OUT',
	LOADING_COMPANIES: 'LOGIN_LOADING_COMPANIES',
	NO_USERS_FOUND: 'LOGIN_NO_USERS_FOUND',
	COMPANY_NAME_HELP: 'LOGIN_COMPANY_NAME_HELP'
} as const;

export const HOME = {
	TITLE: 'HOME_TITLE',
	DESCRIPTION: 'HOME_DESCRIPTION'
} as const;

export const MENU = {
	HOME: 'MENU_HOME',
	LANGUAGE: 'MENU_LANGUAGE',
	COMPANY: 'MENU_COMPANY',
	USER_MENU: 'MENU_USER_MENU',
	SIGN_OUT: 'MENU_SIGN_OUT'
} as const;

export const CARD = {
	NO_RECORD: 'CARD_NO_RECORD',
	SELECT_OR_CREATE: 'CARD_SELECT_OR_CREATE',
	CREATE_NEW: 'CARD_CREATE_NEW',
	CLEAR_FORM: 'CARD_CLEAR_FORM'
} as const;

export const LIST = {
	SEARCH_PLACEHOLDER: 'LIST_SEARCH_PLACEHOLDER',
	CLEAR_SEARCH: 'LIST_CLEAR_SEARCH',
	TOGGLE_ROW_NUMBERS: 'LIST_TOGGLE_ROW_NUMBERS',
	CUSTOMIZE_COLUMNS: 'LIST_CUSTOMIZE_COLUMNS',
	TOGGLE_FILTERS: 'LIST_TOGGLE_FILTERS',
	CUSTOMIZE: 'LIST_CUSTOMIZE',
	FILTER: 'LIST_FILTER'
} as const;

export const FILTER = {
	TITLE: 'FILTER_TITLE',
	VIEWS: 'FILTER_VIEWS',
	ALL: 'FILTER_ALL',
	SAVE_VIEW: 'FILTER_SAVE_VIEW',
	LIST_BY: 'FILTER_LIST_BY',
	ADD: 'FILTER_ADD',
	SELECT_FIELD: 'FILTER_SELECT_FIELD',
	ENTER_EXPRESSION: 'FILTER_ENTER_EXPRESSION',
	DELETE_VIEW: 'FILTER_DELETE_VIEW',
	DELETE_VIEW_CONFIRM: 'FILTER_DELETE_VIEW_CONFIRM',
	RENAME: 'FILTER_RENAME',
	DELETE: 'FILTER_DELETE'
} as const;

export const MODAL = {
	FULLSCREEN: 'MODAL_FULLSCREEN',
	EXIT_FULLSCREEN: 'MODAL_EXIT_FULLSCREEN',
	OPEN_NEW_WINDOW: 'MODAL_OPEN_NEW_WINDOW',
	CLOSE: 'MODAL_CLOSE'
} as const;

export const PLC = {
	ENTER_USER_ID: 'PLC_ENTER_USER_ID',
	ENTER_PASSWORD: 'PLC_ENTER_PASSWORD',
	COMPANY_NAME_EXAMPLE: 'PLC_COMPANY_NAME_EXAMPLE'
} as const;
