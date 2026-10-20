<script lang="ts">
	import Modal from './Modal.svelte';
	import Button from './Button.svelte';
	import { t, MSG, DLG, BTN } from '$lib/services/i18n.svelte';
	import { getDateFormatPattern, parseLocaleDate, formatDate } from '$lib/utils/fieldHelpers';
	import { currentLanguage } from '$lib/stores/session';

	interface InputField {
		name: string;
		label: string;
		type: string;
		required?: boolean;
		default?: string;
		options?: string[]; // type "option": the choices; the value is the chosen number (1-based)
	}

	interface Props {
		open?: boolean;
		title?: string;
		message?: string;
		progress?: number;
		error?: string;
		confirmMode?: boolean;
		confirmMessage?: string;
		onConfirmResponse?: (response: boolean) => void;
		inputMode?: boolean;
		inputFields?: InputField[];
		onInputResponse?: (values: Record<string, string> | null) => void;
		showCancel?: boolean;
		onCancel?: () => void;
	}

	let {
		open = false,
		title, // default looked up in the template (translations may load after mount)
		message = '',
		progress = 0,
		error = '',
		confirmMode = false,
		confirmMessage = '',
		onConfirmResponse,
		inputMode = false,
		inputFields = [],
		onInputResponse,
		showCancel = false,
		onCancel
	}: Props = $props();

	const displayPercent = $derived(Math.round(progress));

	// The input form gets the focus when it appears (the modal was already open): the chosen
	// option of an option list (arrow keys move it), else the first input
	let inputFormElement = $state<HTMLDivElement | null>(null);
	$effect(() => {
		if (!inputMode || !inputFormElement) return;
		const form = inputFormElement;
		setTimeout(() => {
			const target =
				form.querySelector<HTMLInputElement>('input[type="radio"]:checked') ??
				form.querySelector<HTMLInputElement>('input');
			target?.focus();
		}, 0);
	});

	// Enter in the input form is OK (when the values are valid), Esc is Cancel
	function handleInputKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter') {
			event.preventDefault();
			if (inputValid()) handleInputOk();
		} else if (event.key === 'Escape') {
			event.preventDefault();
			handleInputCancel();
		}
	}

	// Locale for date formatting
	let locale = $state('en-US');
	currentLanguage.subscribe((lang) => { locale = lang; });

	// Input form values
	let inputValues = $state<Record<string, string>>({});

	// Initialize input values when inputMode activates with new fields
	$effect(() => {
		if (inputMode && inputFields.length > 0) {
			const initial: Record<string, string> = {};
			for (const field of inputFields) {
				if (field.type === 'date' && field.default) {
					// Convert ISO default to locale format for display
					initial[field.name] = formatDate(field.default, locale);
				} else {
					initial[field.name] = field.default ?? '';
				}
			}
			inputValues = initial;
		}
	});

	// Check if all required fields have values (and date fields parse correctly)
	const inputValid = $derived(() => {
		if (!inputMode) return false;
		return inputFields.every(f => {
			const val = (inputValues[f.name] ?? '').trim();
			if (f.required && val === '') return false;
			if (f.type === 'date' && val !== '') {
				return parseLocaleDate(val, locale) !== null;
			}
			return true;
		});
	});

	function handleYes() {
		onConfirmResponse?.(true);
	}

	function handleNo() {
		onConfirmResponse?.(false);
	}

	function handleCancel() {
		onCancel?.();
	}

	function handleInputOk() {
		// Convert locale-formatted date values back to ISO before sending
		const result: Record<string, string> = { ...inputValues };
		for (const field of inputFields) {
			if (field.type === 'date' && result[field.name]) {
				const iso = parseLocaleDate(result[field.name], locale);
				if (iso) result[field.name] = iso;
			}
		}
		onInputResponse?.(result);
	}

	function handleInputCancel() {
		onInputResponse?.(null);
	}

	function getInputType(fieldType: string): string {
		switch (fieldType) {
			case 'date': return 'text'; // Use text input with locale placeholder
			case 'number': return 'number';
			default: return 'text';
		}
	}
</script>

<Modal {open}>
	<div class="progress-modal">
		<div class="progress-header">
			<h3 class="progress-title">{confirmMode ? t(DLG.CONFIRM_TITLE) : (title ?? t(MSG.PROCESSING))}</h3>
		</div>

		<div class="progress-body">
			{#if confirmMode}
				<div class="confirm-container">
					<svg class="confirm-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<circle cx="12" cy="12" r="10" />
						<path d="M12 16v-4" />
						<path d="M12 8h.01" />
					</svg>
					<p class="confirm-message">{confirmMessage}</p>
				</div>
				<div class="confirm-buttons">
					<Button variant="secondary" onclick={handleNo}>{t(BTN.NO)}</Button>
					<Button variant="primary" onclick={handleYes}>{t(BTN.YES)}</Button>
				</div>
			{:else if inputMode}
				<!-- svelte-ignore a11y_no_static_element_interactions -->
				<div class="input-form" bind:this={inputFormElement} onkeydown={handleInputKeydown}>
					{#each inputFields as field}
						{#if field.type === 'option'}
							<div class="input-field" role="radiogroup" aria-label={field.label || title}>
								{#if field.label}<span class="input-label">{field.label}</span>{/if}
								{#each field.options ?? [] as option, i}
									<label class="option-choice">
										<input
											type="radio"
											name="input-{field.name}"
											value={String(i + 1)}
											bind:group={inputValues[field.name]}
											ondblclick={() => { inputValues[field.name] = String(i + 1); handleInputOk(); }}
										/>
										<span>{option}</span>
									</label>
								{/each}
							</div>
						{:else}
						<div class="input-field">
							<label class="input-label" for="input-{field.name}">
								{field.label}
								{#if field.required}<span class="required-mark">*</span>{/if}
							</label>
							<input
								id="input-{field.name}"
								type={getInputType(field.type)}
								class="input-control"
								placeholder={field.type === 'date' ? getDateFormatPattern(locale) : undefined}
								bind:value={inputValues[field.name]}
							/>
						</div>
						{/if}
					{/each}
				</div>
				<div class="confirm-buttons">
					<Button variant="secondary" onclick={handleInputCancel}>{t(BTN.CANCEL)}</Button>
					<Button variant="primary" onclick={handleInputOk} disabled={!inputValid()}>{t(BTN.OK)}</Button>
				</div>
			{:else if error}
				<div class="error-container">
					<svg class="error-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<circle cx="12" cy="12" r="10" />
						<line x1="12" y1="8" x2="12" y2="12" />
						<line x1="12" y1="16" x2="12.01" y2="16" />
					</svg>
					<p class="error-message">{error}</p>
				</div>
			{:else}
				{#if message}
					<p class="status-message">{message}</p>
				{/if}

				<div class="progress-container">
					<div class="progress-bar-track">
						<div class="progress-bar-fill" style="width: {progress}%"></div>
					</div>
					<span class="progress-percent">{displayPercent}%</span>
				</div>

				{#if showCancel}
					<div class="cancel-container">
						<Button variant="secondary" onclick={handleCancel}>{t(BTN.CANCEL)}</Button>
					</div>
				{/if}
			{/if}
		</div>
	</div>
</Modal>

<style>
	.progress-modal {
		@apply p-6 min-w-[320px];
	}

	.progress-header {
		@apply mb-4;
	}

	.progress-title {
		@apply text-lg font-semibold text-gray-900;
	}

	:global(.dark) .progress-title {
		color: rgb(var(--gray-100));
	}

	.progress-body {
		@apply space-y-4;
	}

	.status-message {
		@apply text-sm text-gray-600;
		white-space: pre-line; /* a progress window's template has several lines */
	}

	:global(.dark) .status-message {
		color: rgb(var(--gray-400));
	}

	.progress-container {
		@apply flex items-center gap-3;
	}

	.progress-bar-track {
		@apply flex-1 h-3 bg-gray-200 rounded-full overflow-hidden;
	}

	:global(.dark) .progress-bar-track {
		background-color: rgb(var(--gray-700));
	}

	.progress-bar-fill {
		@apply h-full bg-blue-500 rounded-full transition-all duration-150 ease-out;
	}

	.progress-percent {
		@apply text-sm font-medium text-gray-700 tabular-nums w-12 text-right;
	}

	:global(.dark) .progress-percent {
		color: rgb(var(--gray-300));
	}

	.error-container {
		@apply flex items-start gap-3 p-3 bg-red-50 rounded-lg;
	}

	:global(.dark) .error-container {
		background-color: rgba(239, 68, 68, 0.1);
	}

	.error-icon {
		@apply w-5 h-5 text-red-500 flex-shrink-0 mt-0.5;
	}

	.error-message {
		@apply text-sm text-red-700;
	}

	:global(.dark) .error-message {
		color: #fca5a5;
	}

	.confirm-container {
		@apply flex items-start gap-3 p-3 bg-blue-50 rounded-lg;
	}

	:global(.dark) .confirm-container {
		background-color: rgba(0, 131, 143, 0.1);
	}

	.confirm-icon {
		@apply w-5 h-5 text-blue-500 flex-shrink-0 mt-0.5;
	}

	.confirm-message {
		@apply text-sm text-gray-700;
	}

	:global(.dark) .confirm-message {
		color: rgb(var(--gray-300));
	}

	.confirm-buttons {
		@apply flex justify-end gap-3 mt-4;
	}

	.cancel-container {
		@apply flex justify-end mt-4;
	}

	/* Input form styles */
	.input-form {
		@apply space-y-3;
	}

	.input-field {
		@apply flex flex-col gap-1;
	}

	.input-label {
		@apply text-sm font-medium text-gray-700;
	}

	:global(.dark) .input-label {
		color: rgb(var(--gray-300));
	}

	.option-choice {
		@apply flex items-center gap-2 px-2 py-1 text-sm rounded cursor-pointer text-gray-900;
	}

	.option-choice:hover {
		@apply bg-gray-100;
	}

	:global(.dark) .option-choice {
		color: rgb(var(--gray-100));
	}

	:global(.dark) .option-choice:hover {
		background-color: rgb(var(--gray-700));
	}

	.required-mark {
		@apply text-red-500 ml-0.5;
	}

	.input-control {
		@apply w-full px-3 py-2 text-sm border border-gray-300 rounded-md;
		@apply bg-surface text-gray-900;
		@apply focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500;
	}

	:global(.dark) .input-control {
		background-color: rgb(var(--gray-700));
		border-color: rgb(var(--gray-600));
		color: rgb(var(--gray-100));
	}

	:global(.dark) .input-control:focus {
		border-color: rgb(var(--accent-500));
	}
</style>
