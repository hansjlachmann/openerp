// Navigation helper functions for record navigation

export interface NavigationState {
	recordIds: string[];
	currentRecordIndex: number;
}

export interface NavigationActions {
	navigateFirst: () => void;
	navigatePrevious: () => void;
	navigateNext: () => void;
	navigateLast: () => void;
}

/**
 * Creates navigation action functions given the current state and a navigate callback.
 * This pattern allows the same navigation logic to be used with different navigation implementations
 * (e.g., page navigation vs modal record loading).
 *
 * @param getState - Function that returns current navigation state (recordIds and currentRecordIndex)
 * @param navigateToRecord - Callback function to handle navigating to a specific record ID
 * @returns Object with navigateFirst, navigatePrevious, navigateNext, navigateLast functions
 */
export function createNavigationActions(
	getState: () => NavigationState,
	navigateToRecord: (recordId: string) => void
): NavigationActions {
	return {
		navigateFirst: () => {
			const { recordIds } = getState();
			if (recordIds.length > 0) {
				navigateToRecord(recordIds[0]);
			}
		},
		navigatePrevious: () => {
			const { recordIds, currentRecordIndex } = getState();
			if (currentRecordIndex > 0) {
				navigateToRecord(recordIds[currentRecordIndex - 1]);
			}
		},
		navigateNext: () => {
			const { recordIds, currentRecordIndex } = getState();
			if (currentRecordIndex < recordIds.length - 1) {
				navigateToRecord(recordIds[currentRecordIndex + 1]);
			}
		},
		navigateLast: () => {
			const { recordIds } = getState();
			if (recordIds.length > 0) {
				navigateToRecord(recordIds[recordIds.length - 1]);
			}
		}
	};
}

/**
 * Checks if navigation to first/previous record is possible
 */
export function canNavigatePrevious(state: NavigationState, recordIdsLoaded: boolean): boolean {
	return recordIdsLoaded && state.currentRecordIndex > 0;
}

/**
 * Checks if navigation to next/last record is possible
 */
export function canNavigateNext(state: NavigationState, recordIdsLoaded: boolean): boolean {
	return recordIdsLoaded && state.currentRecordIndex >= 0 && state.currentRecordIndex < state.recordIds.length - 1;
}
