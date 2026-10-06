// Drilldowns and card actions open another page; like BC, Esc or the close button returns
// to the page they came from, on the same record. The target page gets the way back as
// ?back=<path of the origin page, with select=<record key>>.

// url with back=<the current page>, selecting selectKey there when the user returns
export function withReturn(url: string, selectKey?: string): string {
	const origin = new URL(window.location.href);
	origin.searchParams.delete('back');
	origin.searchParams.delete('select');
	if (selectKey) origin.searchParams.set('select', selectKey);
	const back = origin.pathname + origin.search;
	return url + (url.includes('?') ? '&' : '?') + 'back=' + encodeURIComponent(back);
}

// The back parameter if it is a path of this app (never an absolute or protocol-relative
// URL, which would let a link send the user to another site)
export function safeReturnUrl(value: string | null | undefined): string | undefined {
	if (!value || !value.startsWith('/') || value.startsWith('//') || value.includes('://') || value.includes('\\')) {
		return undefined;
	}
	return value;
}
