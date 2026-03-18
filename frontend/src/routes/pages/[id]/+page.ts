import type { PageLoad } from './$types';

export const load: PageLoad = ({ params, url }) => {
	const pageId = parseInt(params.id, 10);
	const recordId = url.searchParams.get('record') || undefined;
	const filter = url.searchParams.get('filter') || undefined;

	return {
		pageId,
		recordId,
		filter
	};
};
