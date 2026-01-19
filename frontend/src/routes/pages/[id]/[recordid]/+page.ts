import type { PageLoad } from './$types';

export const load: PageLoad = ({ params }) => {
	const pageId = parseInt(params.id, 10);
	const recordId = params.recordid;

	return {
		pageId,
		recordId
	};
};
