<script lang="ts">
	// Runs a page object — a codeunit, a report or a registered function (run_object) — with a
	// record, through the job API, and shows its dialogs: progress, questions, input / StrMenu /
	// request pages, a page it opens (RunModal) and its message box. Used by card pages; list
	// pages have the same dialogs inline.
	import Modal from './Modal.svelte';
	import ProgressModal from './ProgressModal.svelte';
	import PageRenderer from './pages/PageRenderer.svelte';
	import { startJob } from '$lib/services/jobs';
	import { toast } from '$lib/stores/toast';
	import { t, MSG, ERR, BTN } from '$lib/services/i18n.svelte';
	import type { DialogResult } from '$lib/types/api';
	import type { RequestDataItem } from '$lib/types/pages';

	interface Props {
		// After a run: refresh = the result asked for it (or an async job ran)
		ondone?: (refresh: boolean) => void;
	}
	let { ondone }: Props = $props();

	let progressOpen = $state(false);
	let progressTitle = $state('');
	let progressValue = $state(0);
	let progressMessage = $state('');
	let progressError = $state('');
	let confirmMode = $state(false);
	let confirmMessage = $state('');
	let confirmCallback = $state<((r: boolean) => void) | undefined>(undefined);
	let inputMode = $state(false);
	let inputFields = $state<Array<{ name: string; label: string; type: string; required?: boolean; default?: string; options?: string[] }>>([]);
	let inputDataItems = $state<RequestDataItem[]>([]);
	let optionsCaption = $state('');
	let inputCallback = $state<((v: Record<string, string> | null) => void) | undefined>(undefined);
	let runPage = $state<{
		pageId: number;
		filters: import('$lib/types/api').TableFilter[];
		select?: string;
		lookupMode: boolean;
		respond: (values: Record<string, string> | null) => void;
	} | null>(null);
	let dialog = $state<DialogResult | null>(null);

	/**
	 * Run a run_object: "codeunit:455", "report:675", a function "table:X.F(0)" — with the
	 * record (a card's record)
	 */
	export async function run(runObject: string, record: Record<string, unknown>) {
		const [objectType, rest] = runObject.split(':');
		const isFunction = /^[a-z]+:[^.(]+\.[A-Za-z_]\w*(\(.*\))?$/.test(runObject);
		const isReport = objectType === 'report';
		const id = isFunction ? 1 : parseInt(rest ?? '', 10);
		if (!isFunction && (!['codeunit', 'report'].includes(objectType) || isNaN(id) || id <= 0)) {
			toast.error(t(ERR.UNKNOWN_OBJECT_TYPE, objectType));
			return;
		}

		progressOpen = true;
		progressTitle = t(MSG.PROCESSING);
		progressValue = 0;
		progressMessage = '';
		progressError = '';
		confirmMode = false;
		inputMode = false;
		let refresh = false;
		try {
			const result = await startJob(
				id,
				record,
				{
					onProgress: (event) => {
						confirmMode = false;
						inputMode = false;
						progressValue = event.value;
						if (event.message) progressMessage = event.message;
					},
					onComplete: (event) => {
						confirmMode = false;
						inputMode = false;
						progressValue = 100;
						if (event.message) progressMessage = event.message;
						if (event.data?.refresh) refresh = true;
					},
					onError: (err) => {
						confirmMode = false;
						inputMode = false;
						progressError = err;
					},
					onConfirm: (event, respond) => {
						confirmMode = true;
						confirmMessage = event.message;
						confirmCallback = (response: boolean) => {
							confirmMode = false;
							respond(response);
						};
					},
					onRunPage: (event, respond) => {
						const data = event.data ?? {};
						runPage = {
							pageId: Number(data.page_id),
							filters: (data.filters as import('$lib/types/api').TableFilter[]) ?? [],
							select: (data.select as string) || undefined,
							lookupMode: data.lookup_mode === true,
							respond
						};
					},
					onInputRequest: (event, respond) => {
						inputMode = true;
						inputFields = (event.data?.fields as typeof inputFields) || [];
						inputDataItems = (event.data?.data_items as RequestDataItem[]) || [];
						optionsCaption = (event.data?.options_caption as string) || '';
						if (event.message) progressTitle = event.message;
						inputCallback = (values: Record<string, string> | null) => {
							inputMode = false;
							inputFields = [];
							respond(values);
						};
					},
					onSyncResult: (syncResult) => {
						progressOpen = false;
						refresh = !!syncResult.refresh;
						if (syncResult.success) {
							if (syncResult.dialog) dialog = syncResult.dialog;
							else toast.success(syncResult.message || t(MSG.CODEUNIT_SUCCESS));
						} else {
							toast.error(syncResult.message || t(ERR.CODEUNIT_FAILED));
						}
					}
				},
				undefined,
				isFunction ? 'function' : isReport ? 'report' : 'codeunit',
				isFunction ? runObject : undefined
			);
			if (result && 'job_id' in result) {
				await new Promise((resolve) => setTimeout(resolve, 500));
				progressOpen = false;
				if (!progressError) toast.success(t(MSG.JOB_COMPLETED));
				refresh = true; // an async job may have changed anything
			}
			ondone?.(refresh);
		} catch (err) {
			progressOpen = false;
			toast.error(err instanceof Error ? err.message : t(ERR.CODEUNIT_FAILED));
		}
	}
</script>

<ProgressModal
	open={progressOpen}
	title={progressTitle}
	message={progressMessage}
	progress={progressValue}
	error={progressError}
	confirmMode={confirmMode}
	confirmMessage={confirmMessage}
	onConfirmResponse={confirmCallback}
	inputMode={inputMode}
	{inputFields}
	{inputDataItems}
	{optionsCaption}
	onInputResponse={inputCallback}
/>

<!-- A page the codeunit opened (RunModal) -->
{#if runPage}
	{@const request = runPage}
	<Modal open={true} size="expanded">
		<div class="run-page">
			<PageRenderer
				pageid={request.pageId}
				embedded={true}
				initialFilters={request.filters}
				initialSelect={request.select}
				lookupMode={request.lookupMode}
				onclose={(action, key) => {
					runPage = null;
					request.respond(action === 'Cancel' ? null : { action, key: key ?? '' });
				}}
			/>
		</div>
	</Modal>
{/if}

<!-- The result's message box (Esc closes it) -->
{#if dialog}
	{@const d = dialog}
	<Modal open={true} onclose={() => (dialog = null)}>
		<div class="p-6">
			<h3 class="text-lg font-semibold text-gray-900 dark:text-white">{d.title}</h3>
			<p class="mt-2 text-gray-600 dark:text-gray-300" class:text-red-600={d.type === 'error'}>{d.message}</p>
			<div class="mt-6 flex justify-end">
				<button
					class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
					onclick={() => (dialog = null)}
				>
					{t(BTN.OK)}
				</button>
			</div>
		</div>
	</Modal>
{/if}

<style>
	.run-page {
		height: 75vh;
		display: flex;
		flex-direction: column;
	}
</style>
