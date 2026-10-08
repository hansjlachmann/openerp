import type { PageLoad } from './$types';
import { parseNavigationQuery } from '$lib/utils/recordNavigation';

export const load: PageLoad = ({ params, url }) => {
	const pageId = parseInt(params.id, 10);
	const recordId = params.recordid;
	// Opened from a list: card navigation moves through the list's records (filters, search, sort)
	const navigationQuery = parseNavigationQuery(url.searchParams.get('nav'));

	return {
		pageId,
		recordId,
		navigationQuery
	};
};
