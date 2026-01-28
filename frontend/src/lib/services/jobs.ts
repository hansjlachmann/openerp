/**
 * Jobs Service - Handles background job execution with progress tracking
 * Uses Server-Sent Events (SSE) for real-time progress updates
 */

export interface ProgressEvent {
	job_id: string;
	field: number;
	value: number;
	message: string;
	completed: boolean;
	error: string;
	timestamp: number;
	event_type?: string;
	confirm_id?: string;
}

export interface DialogResult {
	title: string;
	message: string;
	type: 'info' | 'success' | 'warning' | 'error';
}

export interface SyncJobResult {
	success: boolean;
	message: string;
	data?: Record<string, unknown>;
	dialog?: DialogResult;
}

export interface JobCallbacks {
	onProgress?: (event: ProgressEvent) => void;
	onComplete?: (event: ProgressEvent) => void;
	onError?: (error: string) => void;
	onSyncResult?: (result: SyncJobResult) => void;
	onConfirm?: (event: ProgressEvent, respond: (response: boolean) => void) => void;
}

/**
 * Start a job (codeunit execution)
 * The backend decides whether to run sync or async based on UsesProgress()
 * Returns a promise that resolves when the job completes
 */
export async function startJob(
	codeunitId: number,
	record: Record<string, unknown>,
	callbacks: JobCallbacks = {}
): Promise<ProgressEvent | SyncJobResult | null> {
	// Start the job
	const response = await fetch('/api/jobs/start', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({
			codeunit_id: codeunitId,
			record
		})
	});

	if (!response.ok) {
		const error = await response.json();
		callbacks.onError?.(error.error || 'Failed to start job');
		throw new Error(error.error || 'Failed to start job');
	}

	const result = await response.json();

	// Check if this is a sync result (no job_id) or async job (has job_id)
	if (result.data.job_id) {
		// Async job - connect to SSE for progress updates
		const jobId = result.data.job_id;
		return new Promise((resolve, reject) => {
			const eventSource = new EventSource(`/api/jobs/${jobId}/events`);
			let lastEvent: ProgressEvent | null = null;

			eventSource.addEventListener('connected', () => {
				// Connection established
			});

			eventSource.addEventListener('progress', (e: MessageEvent) => {
				try {
					const event: ProgressEvent = JSON.parse(e.data);
					lastEvent = event;

					if (event.completed) {
						eventSource.close();
						if (event.error) {
							callbacks.onError?.(event.error);
							reject(new Error(event.error));
						} else {
							callbacks.onComplete?.(event);
							resolve(event);
						}
					} else if (event.event_type === 'confirm') {
						// Handle confirm dialog
						const respond = async (response: boolean) => {
							try {
								await fetch(`/api/jobs/${jobId}/confirm`, {
									method: 'POST',
									headers: { 'Content-Type': 'application/json' },
									body: JSON.stringify({
										confirm_id: event.confirm_id,
										response
									})
								});
							} catch (err) {
								console.error('Failed to send confirm response:', err);
							}
						};
						callbacks.onConfirm?.(event, respond);
					} else {
						callbacks.onProgress?.(event);
					}
				} catch (err) {
					console.error('Failed to parse progress event:', err);
				}
			});

			eventSource.addEventListener('close', () => {
				eventSource.close();
				if (lastEvent) {
					resolve(lastEvent);
				} else {
					resolve(null);
				}
			});

			eventSource.onerror = () => {
				eventSource.close();
				const error = 'Connection lost';
				callbacks.onError?.(error);
				reject(new Error(error));
			};
		});
	} else {
		// Sync result - return immediately
		const syncResult: SyncJobResult = {
			success: result.data.success,
			message: result.data.message,
			data: result.data.data,
			dialog: result.data.dialog
		};
		callbacks.onSyncResult?.(syncResult);
		return syncResult;
	}
}

/**
 * Run a job with progress modal support
 * This is a higher-level function that manages the progress state
 */
export function createJobRunner() {
	let isRunning = $state(false);
	let progress = $state(0);
	let message = $state('');
	let error = $state('');
	let title = $state('Processing...');

	async function run(
		codeunitId: number,
		record: Record<string, unknown>,
		jobTitle: string = 'Processing...'
	): Promise<boolean> {
		isRunning = true;
		progress = 0;
		message = '';
		error = '';
		title = jobTitle;

		try {
			await startJob(codeunitId, record, {
				onProgress: (event) => {
					progress = event.value;
					if (event.message) {
						message = event.message;
					}
				},
				onComplete: (event) => {
					progress = 100;
					if (event.message) {
						message = event.message;
					}
				},
				onError: (err) => {
					error = err;
				}
			});

			// Keep modal open briefly to show 100%
			await new Promise((resolve) => setTimeout(resolve, 500));
			isRunning = false;
			return !error;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Unknown error';
			// Keep modal open to show error
			await new Promise((resolve) => setTimeout(resolve, 2000));
			isRunning = false;
			return false;
		}
	}

	function close() {
		isRunning = false;
	}

	return {
		get isRunning() {
			return isRunning;
		},
		get progress() {
			return progress;
		},
		get message() {
			return message;
		},
		get error() {
			return error;
		},
		get title() {
			return title;
		},
		run,
		close
	};
}
