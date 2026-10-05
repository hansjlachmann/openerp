import { writable } from 'svelte/store';

// True while the Switch Company dialog (Ctrl+O, NAV Classic) is open. Pages with
// window-level key handlers (ListPage) check it so Escape/arrows reach the dialog
// instead of acting on the page underneath.
export const companySwitchOpen = writable(false);
