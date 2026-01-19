import { writable } from 'svelte/store';

export interface BreadcrumbItem {
	label: string;
	href: string;
	pageId?: number;
	recordId?: string;
}

interface BreadcrumbState {
	items: BreadcrumbItem[];
	homeLabel: string;
}

function createBreadcrumbStore() {
	const { subscribe, set, update } = writable<BreadcrumbState>({
		items: [],
		homeLabel: 'Home'
	});

	return {
		subscribe,

		// Set the translated home label
		setHomeLabel(label: string) {
			update(state => ({ ...state, homeLabel: label }));
		},

		// Set breadcrumbs for a list page
		setListPage(pageId: number, pageCaption: string, homeLabel?: string) {
			update(state => ({
				homeLabel: homeLabel || state.homeLabel,
				items: [
					{ label: homeLabel || state.homeLabel, href: '/' },
					{ label: pageCaption, href: `/pages/${pageId}`, pageId }
				]
			}));
		},

		// Set breadcrumbs for a card page (with parent list)
		setCardPage(
			pageId: number,
			pageCaption: string,
			recordId: string | undefined,
			recordLabel: string | undefined,
			listPageId?: number,
			listPageCaption?: string,
			homeLabel?: string
		) {
			update(state => {
				const home = homeLabel || state.homeLabel;
				const items: BreadcrumbItem[] = [{ label: home, href: '/' }];

				// Add list page if available
				if (listPageId && listPageCaption) {
					items.push({
						label: listPageCaption,
						href: `/pages/${listPageId}`,
						pageId: listPageId
					});
				}

				// Add card page
				const cardLabel = recordLabel ? `${pageCaption} (${recordLabel})` : pageCaption;
				items.push({
					label: cardLabel,
					href: recordId ? `/pages/${pageId}/${recordId}` : `/pages/${pageId}`,
					pageId,
					recordId
				});

				return { homeLabel: home, items };
			});
		},

		// Clear breadcrumbs (for home page)
		clear() {
			update(state => ({
				...state,
				items: [{ label: state.homeLabel, href: '/' }]
			}));
		},

		// Set custom breadcrumbs
		setCustom(items: BreadcrumbItem[]) {
			update(state => ({
				...state,
				items: [{ label: state.homeLabel, href: '/' }, ...items]
			}));
		}
	};
}

export const breadcrumb = createBreadcrumbStore();
