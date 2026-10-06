import type { PageLoad } from './$types';
import { safeReturnUrl } from '$lib/utils/returnUrl';

export const load: PageLoad = ({ params, url }) => {
	const pageId = parseInt(params.id, 10);
	const recordId = url.searchParams.get('record') || undefined;
	const filter = url.searchParams.get('filter') || undefined;
	// Opened from a drilldown or card action: where Esc / close returns to, and the record
	// to select when this page is the one returned to
	const returnUrl = safeReturnUrl(url.searchParams.get('back'));
	const select = url.searchParams.get('select') || undefined;

	return {
		pageId,
		recordId,
		filter,
		returnUrl,
		select
	};
};
