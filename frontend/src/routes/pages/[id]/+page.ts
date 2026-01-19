import type { PageLoad } from './$types';

export const load: PageLoad = ({ params, url }) => {
	const pageId = parseInt(params.id, 10);
	const recordId = url.searchParams.get('record') || undefined;

	return {
		pageId,
		recordId
	};
};
