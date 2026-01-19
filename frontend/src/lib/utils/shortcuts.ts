// Keyboard shortcut utility for BC/NAV-style keyboard navigation

export type ShortcutHandler = (event: KeyboardEvent) => void | boolean;

export interface ShortcutMap {
	[key: string]: ShortcutHandler;
}

// Normalize shortcut key names to a consistent format
// This handles common aliases like "Esc" -> "Escape"
function normalizeKeyName(key: string): string {
	const aliases: Record<string, string> = {
		'Esc': 'Escape',
		'Del': 'Delete',
		'Ins': 'Insert',
		'PgUp': 'PageUp',
		'PgDn': 'PageDown',
		'PgDown': 'PageDown',
	};
	return aliases[key] || key;
}

// Parse keyboard event to shortcut string (e.g., "Ctrl+N", "F5")
export function getShortcutKey(event: KeyboardEvent): string {
	const parts: string[] = [];

	if (event.ctrlKey || event.metaKey) parts.push('Ctrl');
	if (event.altKey) parts.push('Alt');
	if (event.shiftKey) parts.push('Shift');

	// Normalize key name
	let key = event.key;
	if (key === ' ') key = 'Space';
	if (key.length === 1) key = key.toUpperCase();

	parts.push(key);

	return parts.join('+');
}

// Normalize a shortcut string from page definitions
// Converts aliases like "Esc" to "Escape" for consistent matching
export function normalizeShortcut(shortcut: string): string {
	const parts = shortcut.split('+');
	const normalizedParts = parts.map(part => normalizeKeyName(part));
	return normalizedParts.join('+');
}

// Svelte action for keyboard shortcuts
export function shortcuts(node: HTMLElement, shortcutMap: ShortcutMap) {
	function handleKeydown(event: KeyboardEvent) {
		const shortcutKey = getShortcutKey(event);
		const handler = shortcutMap[shortcutKey];

		if (handler) {
			try {
				const result = handler(event);
				// Prevent default unless handler returns false
				if (result !== false) {
					event.preventDefault();
					event.stopPropagation();
				}
			} catch (err) {
				console.error('Error in shortcut handler:', err);
				// Still prevent default to avoid unexpected behavior
				event.preventDefault();
				event.stopPropagation();
			}
		}
	}

	node.addEventListener('keydown', handleKeydown);

	return {
		update(newShortcutMap: ShortcutMap) {
			shortcutMap = newShortcutMap;
		},
		destroy() {
			node.removeEventListener('keydown', handleKeydown);
		}
	};
}

// Common BC/NAV shortcuts
export const commonShortcuts = {
	NEW: 'Ctrl+N',
	EDIT: 'Ctrl+E',
	DELETE: 'Ctrl+D',
	SAVE: 'Ctrl+S',
	FIND: 'Ctrl+F',
	REFRESH: 'F5',
	FIRST: 'Ctrl+Home',
	LAST: 'Ctrl+End',
	NEXT: 'PageDown',
	PREVIOUS: 'PageUp',
	RENAME: 'F2',
	CANCEL: 'Escape',
	STATISTICS: 'Ctrl+F7',
	CLOSE: 'Alt+F4'
};

// Helper to create shortcut map from actions
export function createShortcutMap(actions: {
	onNew?: () => void;
	onEdit?: () => void;
	onDelete?: () => void;
	onSave?: () => void;
	onFind?: () => void;
	onRefresh?: () => void;
	onFirst?: () => void;
	onLast?: () => void;
	onNext?: () => void;
	onPrevious?: () => void;
	onRename?: () => void;
	onCancel?: () => void;
	onClose?: () => void;
}): ShortcutMap {
	const map: ShortcutMap = {};

	if (actions.onNew) map[commonShortcuts.NEW] = actions.onNew;
	if (actions.onEdit) map[commonShortcuts.EDIT] = actions.onEdit;
	if (actions.onDelete) map[commonShortcuts.DELETE] = actions.onDelete;
	if (actions.onSave) map[commonShortcuts.SAVE] = actions.onSave;
	if (actions.onFind) map[commonShortcuts.FIND] = actions.onFind;
	if (actions.onRefresh) map[commonShortcuts.REFRESH] = actions.onRefresh;
	if (actions.onFirst) map[commonShortcuts.FIRST] = actions.onFirst;
	if (actions.onLast) map[commonShortcuts.LAST] = actions.onLast;
	if (actions.onNext) map[commonShortcuts.NEXT] = actions.onNext;
	if (actions.onPrevious) map[commonShortcuts.PREVIOUS] = actions.onPrevious;
	if (actions.onRename) map[commonShortcuts.RENAME] = actions.onRename;
	if (actions.onCancel) map[commonShortcuts.CANCEL] = actions.onCancel;
	if (actions.onClose) map[commonShortcuts.CLOSE] = actions.onClose;

	return map;
}
