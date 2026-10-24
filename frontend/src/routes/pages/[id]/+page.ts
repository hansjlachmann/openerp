import type { PageLoad } from './$types';
import { safeReturnUrl } from '$lib/utils/returnUrl';

export const load: PageLoad = ({ params, url }) => {
	const pageId = parseInt(params.id, 10);
	const recordId = url.searchParams.get('record') || undefined;
	const filter = url.searchParams.getAll('filter'); // field=expression, one per filter
	// Opened from a drilldown or card action: where Esc / close returns to, and the record
	// to select when this page is the one returned to
	const returnUrl = safeReturnUrl(url.searchParams.get('back'));
	const select = url.searchParams.get('select') || undefined;
	// Returning to a list whose modal card was open on the selected record: open it again
	const openCard = url.searchParams.get('card') === '1';
	// Opened from a lookup's New: the list starts a new record ready to type
	const startNew = url.searchParams.get('new') === '1';

	// The page's own parameters (a journal's template and batch, for its OnOpenPage trigger):
	// every other query parameter
	const reserved = new Set(['record', 'filter', 'back', 'select', 'card', 'new', 'nav']);
	const openParams: Record<string, string> = {};
	for (const [name, value] of url.searchParams) {
		if (!reserved.has(name)) openParams[name] = value;
	}

	return {
		pageId,
		openParams,
		recordId,
		filter,
		returnUrl,
		select,
		openCard,
		startNew
	};
};
