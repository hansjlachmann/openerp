<script lang="ts">
	import { session } from '$stores/session';
</script>

<div class="container mx-auto px-4 py-8">
	<div class="max-w-4xl mx-auto">
		<div class="card">
			<div class="card-header">
				<h1 class="text-3xl font-bold text-nav-blue">OpenERP - Business Central Style ERP</h1>
			</div>
			<div class="card-body">
				<div class="space-y-6">
					<div>
						<h2 class="text-xl font-semibold mb-3">Welcome to OpenERP Web</h2>
						<p class="text-gray-600">
							A modern, fast, and keyboard-driven ERP system inspired by Microsoft Dynamics 365
							Business Central / Dynamics NAV.
						</p>
					</div>

					<div class="border-t border-gray-200 pt-6">
						<h3 class="text-lg font-semibold mb-3">Session Information</h3>
						<dl class="grid grid-cols-2 gap-4">
							<div>
								<dt class="text-sm font-medium text-gray-500">Database</dt>
								<dd class="mt-1 text-sm text-gray-900">{$session.database || 'Not connected'}</dd>
							</div>
							<div>
								<dt class="text-sm font-medium text-gray-500">Company</dt>
								<dd class="mt-1 text-sm text-gray-900">{$session.company || 'None selected'}</dd>
							</div>
							<div>
								<dt class="text-sm font-medium text-gray-500">User</dt>
								<dd class="mt-1 text-sm text-gray-900">{$session.userName || 'Not logged in'}</dd>
							</div>
							<div>
								<dt class="text-sm font-medium text-gray-500">Language</dt>
								<dd class="mt-1 text-sm text-gray-900">{$session.language || 'en-US'}</dd>
							</div>
						</dl>
					</div>

					<div class="border-t border-gray-200 pt-6">
						<h3 class="text-lg font-semibold mb-3">Quick Links</h3>
						<div class="grid grid-cols-2 gap-4">
							<a
								href="/pages/22"
								class="block p-4 border border-gray-300 rounded-lg hover:border-primary-500 hover:bg-primary-50 transition-colors"
							>
								<div class="font-semibold text-nav-blue">Customer List (YAML)</div>
								<div class="text-sm text-gray-600 mt-1">View and manage customers - YAML-driven page</div>
							</a>
							<a
								href="/pages/26"
								class="block p-4 border border-gray-300 rounded-lg hover:border-primary-500 hover:bg-primary-50 transition-colors"
							>
								<div class="font-semibold text-nav-blue">Payment Terms (YAML)</div>
								<div class="text-sm text-gray-600 mt-1">Manage payment terms - YAML-driven page</div>
							</a>
							<a
								href="/customers"
								class="block p-4 border border-gray-300 rounded-lg hover:border-primary-500 hover:bg-primary-50 transition-colors"
							>
								<div class="font-semibold text-nav-blue">Customer List (Old)</div>
								<div class="text-sm text-gray-600 mt-1">Original hardcoded customer list</div>
							</a>
						</div>
					</div>

					<div class="border-t border-gray-200 pt-6">
						<h3 class="text-lg font-semibold mb-3">Keyboard Shortcuts</h3>
						<div class="bg-gray-50 p-4 rounded-lg">
							<table class="w-full text-sm">
								<tbody class="space-y-2">
									<tr>
										<td class="font-mono text-gray-700 py-1">Ctrl+N</td>
										<td class="text-gray-600 py-1">New record</td>
									</tr>
									<tr>
										<td class="font-mono text-gray-700 py-1">Ctrl+E</td>
										<td class="text-gray-600 py-1">Edit record</td>
									</tr>
									<tr>
										<td class="font-mono text-gray-700 py-1">Ctrl+D</td>
										<td class="text-gray-600 py-1">Delete record</td>
									</tr>
									<tr>
										<td class="font-mono text-gray-700 py-1">Ctrl+F</td>
										<td class="text-gray-600 py-1">Find / Filter</td>
									</tr>
									<tr>
										<td class="font-mono text-gray-700 py-1">F5</td>
										<td class="text-gray-600 py-1">Refresh</td>
									</tr>
									<tr>
										<td class="font-mono text-gray-700 py-1">Ctrl+Home/End</td>
										<td class="text-gray-600 py-1">First / Last record</td>
									</tr>
								</tbody>
							</table>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</div>
