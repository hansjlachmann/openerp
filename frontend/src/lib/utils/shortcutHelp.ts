// Content of the Keyboard Shortcuts help window (/help/shortcuts). Key names are shown as
// written; titles and descriptions are translation keys (translations/{lang}/messages.yaml).
// Keep this in step with the keyboard tables in CLAUDE.md when shortcuts change.

export interface ShortcutHelpItem {
	keys: string[]; // alternatives, e.g. ['Alt+N', 'Ctrl+Insert']
	description: string; // translation key
}

export interface ShortcutHelpSection {
	title: string; // translation key
	items: ShortcutHelpItem[];
}

export const SHORTCUT_HELP_URL = '/help/shortcuts';
export const SHORTCUT_HELP_WINDOW = 'openerp-shortcuts';

export const shortcutHelpSections: ShortcutHelpSection[] = [
	{
		title: 'HELP_SEC_GENERAL',
		items: [
			{ keys: ['Ctrl+O'], description: 'HELP_SWITCH_COMPANY' },
			{ keys: ['Esc'], description: 'HELP_ESCAPE' }
		]
	},
	{
		title: 'HELP_SEC_MENU',
		items: [
			{ keys: ['↑', '↓', '←', '→'], description: 'HELP_MENU_ARROWS' },
			{ keys: ['Home', 'End'], description: 'HELP_MENU_HOME_END' },
			{ keys: ['Enter'], description: 'HELP_MENU_OPEN' }
		]
	},
	{
		title: 'HELP_SEC_LIST',
		items: [
			{ keys: ['↑', '↓'], description: 'HELP_LIST_ROWS' },
			{ keys: ['Home', 'End', 'Ctrl+Home', 'Ctrl+End'], description: 'HELP_LIST_FIRST_LAST' },
			{ keys: ['PgUp', 'PgDn'], description: 'HELP_LIST_PAGE' },
			{ keys: ['Enter'], description: 'HELP_LIST_OPEN' },
			{ keys: ['F2'], description: 'HELP_LIST_EDIT_CELLS' },
			{ keys: ['Ctrl+E'], description: 'HELP_LIST_EDIT' },
			{ keys: ['Alt+N', 'Ctrl+Insert'], description: 'HELP_LIST_NEW' },
			{ keys: ['Ctrl+D'], description: 'HELP_LIST_DELETE' },
			{ keys: ['Ctrl+F'], description: 'HELP_LIST_SEARCH' },
			{ keys: ['F5'], description: 'HELP_LIST_REFRESH' },
			{ keys: ['Esc'], description: 'HELP_LIST_ESCAPE' }
		]
	},
	{
		title: 'HELP_SEC_CELLS',
		items: [
			{ keys: ['↑', '↓', '←', '→'], description: 'HELP_CELL_ARROWS' },
			{ keys: ['Tab', 'Shift+Tab'], description: 'HELP_CELL_TAB' },
			{ keys: ['Enter'], description: 'HELP_CELL_ENTER' },
			{ keys: ['PgUp', 'PgDn'], description: 'HELP_CELL_PAGE' },
			{ keys: ['F2'], description: 'HELP_CELL_F2' },
			{ keys: ['F8'], description: 'HELP_CELL_F8' },
			{ keys: ['Delete'], description: 'HELP_CELL_DELETE' },
			{ keys: ['Backspace'], description: 'HELP_CELL_BACKSPACE' },
			{ keys: ['Ctrl+C', 'Ctrl+V'], description: 'HELP_CELL_COPY_PASTE' },
			{ keys: ['Space'], description: 'HELP_CELL_SPACE' },
			{ keys: ['Esc'], description: 'HELP_CELL_ESCAPE' }
		]
	},
	{
		title: 'HELP_SEC_CARD',
		items: [
			{ keys: ['Tab', 'Enter'], description: 'HELP_CARD_NEXT' },
			{ keys: ['Shift+Tab', 'Shift+Enter'], description: 'HELP_CARD_PREV' },
			{ keys: ['Ctrl+↑', 'Ctrl+↓'], description: 'HELP_CARD_PREV_NEXT_RECORD' },
			{ keys: ['Ctrl+Home', 'Ctrl+End'], description: 'HELP_CARD_FIRST_LAST_RECORD' },
			{ keys: ['Esc'], description: 'HELP_CARD_ESCAPE' }
		]
	},
	{
		title: 'HELP_SEC_LOOKUP',
		items: [
			{ keys: ['↓', 'Alt+↓', 'F4'], description: 'HELP_LOOKUP_OPEN' },
			{ keys: ['↑', '↓'], description: 'HELP_LOOKUP_MOVE' },
			{ keys: ['Enter'], description: 'HELP_LOOKUP_PICK' },
			{ keys: ['Esc'], description: 'HELP_LOOKUP_ESCAPE' }
		]
	}
];
