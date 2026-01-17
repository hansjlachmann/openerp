// API response handling utilities

import type { ApiResponse } from '$types/api';

/**
 * Handle API response - checks for HTTP errors and API-level errors
 * @param response - The fetch Response object
 * @param errorContext - Context for error messages (e.g., "list customers")
 * @returns The parsed response data
 * @throws Error if response is not ok or API returns error
 */
export async function handleApiResponse<T>(
	response: Response,
	errorContext: string
): Promise<T> {
	if (!response.ok) {
		// Try to get error from response body
		try {
			const result: ApiResponse = await response.json();
			throw new Error(result.error || `Failed to ${errorContext}: ${response.statusText}`);
		} catch (e) {
			if (e instanceof Error && e.message !== `Failed to ${errorContext}: ${response.statusText}`) {
				throw e;
			}
			throw new Error(`Failed to ${errorContext}: ${response.statusText}`);
		}
	}

	const result: ApiResponse<T> = await response.json();
	if (!result.success) {
		throw new Error(result.error || `Failed to ${errorContext}`);
	}

	return result.data as T;
}

/**
 * Handle API response that returns void (no data expected)
 */
export async function handleApiResponseVoid(
	response: Response,
	errorContext: string
): Promise<void> {
	if (!response.ok) {
		try {
			const result: ApiResponse = await response.json();
			throw new Error(result.error || `Failed to ${errorContext}: ${response.statusText}`);
		} catch (e) {
			if (e instanceof Error && e.message !== `Failed to ${errorContext}: ${response.statusText}`) {
				throw e;
			}
			throw new Error(`Failed to ${errorContext}: ${response.statusText}`);
		}
	}

	const result: ApiResponse = await response.json();
	if (!result.success) {
		throw new Error(result.error || `Failed to ${errorContext}`);
	}
}

/**
 * Handle API response that returns the full ApiResponse (for auth endpoints)
 */
export async function handleApiResponseFull<T = any>(
	response: Response,
	errorContext: string
): Promise<ApiResponse<T>> {
	if (!response.ok) {
		try {
			const result: ApiResponse<T> = await response.json();
			throw { status: response.status, message: result.error || `Failed to ${errorContext}` };
		} catch (e) {
			if (e && typeof e === 'object' && 'status' in e) {
				throw e;
			}
			throw { status: response.status, message: `Failed to ${errorContext}` };
		}
	}

	return await response.json();
}

/**
 * Result type for API calls that need both data and captions
 */
export interface DataWithCaptions<T> {
	data: T;
	captions?: {
		table?: string;
		fields?: Record<string, string>;
		options?: Record<string, Record<string, string>>;
	};
}

/**
 * Handle API response returning data with captions (for table operations that need option values)
 */
export async function handleApiResponseWithCaptions<T>(
	response: Response,
	errorContext: string
): Promise<DataWithCaptions<T>> {
	if (!response.ok) {
		try {
			const result: ApiResponse = await response.json();
			throw new Error(result.error || `Failed to ${errorContext}: ${response.statusText}`);
		} catch (e) {
			if (e instanceof Error && e.message !== `Failed to ${errorContext}: ${response.statusText}`) {
				throw e;
			}
			throw new Error(`Failed to ${errorContext}: ${response.statusText}`);
		}
	}

	const result: ApiResponse<T> = await response.json();
	if (!result.success) {
		throw new Error(result.error || `Failed to ${errorContext}`);
	}

	return {
		data: result.data as T,
		captions: result.captions
	};
}
