<script lang="ts">
	import { goto } from '$app/navigation';
	import { api } from '$lib/services/api';
	import { currentUser } from '$lib/stores/user';
	import { session } from '$stores/session';
	import { toast } from '$lib/stores/toast';
	import { onMount } from 'svelte';

	let userID = $state('');
	let password = $state('');
	let company = $state('');
	let companies = $state<string[]>([]);
	let error = $state('');
	let loading = $state(false);
	let needsInitialSetup = $state(false);
	let setupMode = $state(false);
	let showNewCompanyForm = $state(false);
	let newCompanyName = $state('');

	// Setup form fields
	let setupUserID = $state('');
	let setupUserName = $state('');
	let setupEmail = $state('');
	let setupPassword = $state('');
	let setupPasswordConfirm = $state('');

	onMount(async () => {
		// Load companies
		try {
			const response = await api.listCompanies();
			if (response.success && response.data) {
				companies = response.data;
				// Set default company if available
				if (companies.length > 0) {
					company = companies[0];
				}
			}
		} catch (err) {
			console.error('Failed to load companies:', err);
		}

		// Check if we're already logged in
		try {
			const response = await api.getCurrentUser();
			if (response.success) {
				// Already logged in, redirect to home
				goto('/');
			}
		} catch (err) {
			// Not logged in, check if we need initial setup
			// Try to detect if no users exist by attempting to create initial user with invalid data
			// This will tell us if users already exist
			try {
				const testResponse = await fetch('/api/auth/init', {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ user_id: '', user_name: '', password: '' })
				});

				if (testResponse.status === 403) {
					// Users already exist
					needsInitialSetup = false;
				} else {
					// No users exist, show setup
					needsInitialSetup = true;
					setupMode = true;
				}
			} catch (err) {
				// Assume users exist if we can't check
				needsInitialSetup = false;
			}
		}
	});

	async function handleLogin() {
		error = '';
		if (!userID || !password) {
			error = 'Please enter both User ID and Password';
			return;
		}

		if (!company) {
			error = 'Please select a company';
			return;
		}

		loading = true;
		try {
			const response = await api.login(userID, password, company);
			if (response.success) {
				// Store user info in store (also saves to localStorage)
				currentUser.setUser(response.data);
				// Re-initialize session to get updated language from backend
				await session.initialize();
				toast.success('Welcome back, ' + (response.data.user_name || userID));
				// Redirect to home
				goto('/');
			} else {
				error = response.error || 'Login failed';
				toast.error(error);
				// Check if it's because no users exist
				if (error.includes('Invalid credentials')) {
					// Try to detect if this is the initial setup scenario
					needsInitialSetup = true;
				}
			}
		} catch (err: any) {
			if (err.status === 401) {
				error = 'Invalid credentials';
			} else {
				error = 'An error occurred. Please try again.';
			}
			toast.error(error);
		} finally {
			loading = false;
		}
	}

	async function handleInitialSetup() {
		error = '';

		// Validation
		if (!setupUserID || !setupUserName || !setupPassword) {
			error = 'Please fill in all required fields';
			return;
		}

		if (setupPassword !== setupPasswordConfirm) {
			error = 'Passwords do not match';
			return;
		}

		if (setupPassword.length < 6) {
			error = 'Password must be at least 6 characters';
			return;
		}

		loading = true;
		try {
			const response = await api.createInitialUser({
				user_id: setupUserID,
				user_name: setupUserName,
				email: setupEmail,
				password: setupPassword
			});

			if (response.success) {
				toast.success('Initial user created successfully');
				// User created, now log in
				userID = setupUserID;
				password = setupPassword;
				setupMode = false;
				needsInitialSetup = false;
				await handleLogin();
			} else {
				error = response.error || 'Failed to create user';
				toast.error(error);
			}
		} catch (err: any) {
			error = err.message || 'An error occurred during setup';
			toast.error(error);
		} finally {
			loading = false;
		}
	}

	function handleKeyPress(event: KeyboardEvent) {
		if (event.key === 'Enter') {
			if (setupMode) {
				handleInitialSetup();
			} else if (showNewCompanyForm) {
				handleCreateCompany();
			} else {
				handleLogin();
			}
		}
	}

	async function handleCreateCompany() {
		error = '';
		if (!newCompanyName.trim()) {
			error = 'Please enter a company name';
			return;
		}

		loading = true;
		try {
			const response = await api.createCompany(newCompanyName.trim());
			if (response.success && response.data) {
				toast.success(`Company "${response.data.name}" created successfully`);
				// Refresh companies list
				const listResponse = await api.listCompanies();
				if (listResponse.success && listResponse.data) {
					companies = listResponse.data;
					company = response.data.name;
				}
				showNewCompanyForm = false;
				newCompanyName = '';
			} else {
				error = response.error || 'Failed to create company';
				toast.error(error);
			}
		} catch (err: any) {
			error = err.message || 'An error occurred';
			toast.error(error);
		} finally {
			loading = false;
		}
	}
</script>

<div class="login-container">
	<div class="login-card">
		<div class="login-header">
			<h1>OpenERP</h1>
			<p>{setupMode ? 'Initial Setup' : 'Sign In'}</p>
		</div>

		{#if error}
			<div class="error-message">
				{error}
			</div>
		{/if}

		{#if setupMode}
			<!-- Initial Setup Form -->
			<div class="form-group">
				<label for="setup-userid">User ID *</label>
				<input
					id="setup-userid"
					type="text"
					bind:value={setupUserID}
					placeholder="admin"
					disabled={loading}
					onkeypress={handleKeyPress}
				/>
			</div>

			<div class="form-group">
				<label for="setup-username">Full Name *</label>
				<input
					id="setup-username"
					type="text"
					bind:value={setupUserName}
					placeholder="Administrator"
					disabled={loading}
					onkeypress={handleKeyPress}
				/>
			</div>

			<div class="form-group">
				<label for="setup-email">Email</label>
				<input
					id="setup-email"
					type="email"
					bind:value={setupEmail}
					placeholder="admin@example.com"
					disabled={loading}
					onkeypress={handleKeyPress}
				/>
			</div>

			<div class="form-group">
				<label for="setup-password">Password *</label>
				<input
					id="setup-password"
					type="password"
					bind:value={setupPassword}
					placeholder="••••••••"
					disabled={loading}
					onkeypress={handleKeyPress}
				/>
			</div>

			<div class="form-group">
				<label for="setup-password-confirm">Confirm Password *</label>
				<input
					id="setup-password-confirm"
					type="password"
					bind:value={setupPasswordConfirm}
					placeholder="••••••••"
					disabled={loading}
					onkeypress={handleKeyPress}
				/>
			</div>

			<button class="login-button" onclick={handleInitialSetup} disabled={loading}>
				{loading ? 'Creating...' : 'Create Initial User'}
			</button>

			<button class="secondary-button" onclick={() => { setupMode = false; needsInitialSetup = false; }} disabled={loading}>
				Back to Login
			</button>
		{:else if showNewCompanyForm}
			<!-- New Company Form -->
			<div class="form-group">
				<label for="new-company">Company Name</label>
				<input
					id="new-company"
					type="text"
					bind:value={newCompanyName}
					placeholder="e.g., my-company"
					disabled={loading}
					onkeypress={handleKeyPress}
					autofocus
				/>
				<small class="help-text">Letters, numbers, underscores, and hyphens only</small>
			</div>

			<button class="login-button" onclick={handleCreateCompany} disabled={loading}>
				{loading ? 'Creating...' : 'Create Company'}
			</button>

			<button class="secondary-button" onclick={() => { showNewCompanyForm = false; error = ''; }} disabled={loading}>
				Back to Login
			</button>
		{:else}
			<!-- Login Form -->
			<div class="form-group">
				<label for="company">Company</label>
				<div class="company-row">
					<select
						id="company"
						bind:value={company}
						disabled={loading || companies.length === 0}
						class="company-select"
					>
						{#if companies.length === 0}
							<option value="">Loading companies...</option>
						{:else}
							{#each companies as companyName}
								<option value={companyName}>{companyName}</option>
							{/each}
						{/if}
					</select>
					<button
						class="new-company-btn"
						onclick={() => { showNewCompanyForm = true; error = ''; }}
						disabled={loading}
						title="Create new company"
					>
						+
					</button>
				</div>
			</div>

			<div class="form-group">
				<label for="userid">User ID</label>
				<input
					id="userid"
					type="text"
					bind:value={userID}
					placeholder="Enter your user ID"
					disabled={loading}
					onkeypress={handleKeyPress}
					autofocus
				/>
			</div>

			<div class="form-group">
				<label for="password">Password</label>
				<input
					id="password"
					type="password"
					bind:value={password}
					placeholder="Enter your password"
					disabled={loading}
					onkeypress={handleKeyPress}
				/>
			</div>

			<button class="login-button" onclick={handleLogin} disabled={loading}>
				{loading ? 'Signing in...' : 'Sign In'}
			</button>

			{#if needsInitialSetup}
				<div class="setup-prompt">
					<p>No users found in the system.</p>
					<button class="secondary-button" onclick={() => setupMode = true}>
						Create Initial User
					</button>
				</div>
			{/if}
		{/if}
	</div>
</div>

<style>
	.login-container {
		display: flex;
		align-items: center;
		justify-content: center;
		min-height: 100vh;
		background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
		padding: 1rem;
	}

	:global(.dark) .login-container {
		background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
	}

	.login-card {
		background: white;
		border-radius: 8px;
		box-shadow: 0 10px 40px rgba(0, 0, 0, 0.2);
		padding: 2rem;
		width: 100%;
		max-width: 420px;
	}

	:global(.dark) .login-card {
		background: #1f2937;
		box-shadow: 0 10px 40px rgba(0, 0, 0, 0.5);
	}

	.login-header {
		text-align: center;
		margin-bottom: 2rem;
	}

	.login-header h1 {
		margin: 0 0 0.5rem 0;
		color: #2d3748;
		font-size: 2rem;
	}

	:global(.dark) .login-header h1 {
		color: #f3f4f6;
	}

	.login-header p {
		margin: 0;
		color: #718096;
		font-size: 1rem;
	}

	:global(.dark) .login-header p {
		color: #9ca3af;
	}

	.error-message {
		background-color: #fee;
		border: 1px solid #fcc;
		border-radius: 4px;
		color: #c33;
		padding: 0.75rem;
		margin-bottom: 1rem;
		font-size: 0.875rem;
	}

	:global(.dark) .error-message {
		background-color: #450a0a;
		border-color: #7f1d1d;
		color: #fca5a5;
	}

	.form-group {
		margin-bottom: 1.25rem;
	}

	.form-group label {
		display: block;
		margin-bottom: 0.5rem;
		color: #2d3748;
		font-weight: 500;
		font-size: 0.875rem;
	}

	:global(.dark) .form-group label {
		color: #e5e7eb;
	}

	.form-group input {
		width: 100%;
		padding: 0.75rem;
		border: 1px solid #cbd5e0;
		border-radius: 4px;
		font-size: 1rem;
		transition: border-color 0.15s ease-in-out;
		box-sizing: border-box;
		background-color: white;
		color: #2d3748;
	}

	:global(.dark) .form-group input {
		background-color: #374151;
		border-color: #4b5563;
		color: #f3f4f6;
	}

	.form-group input:focus {
		outline: none;
		border-color: #667eea;
		box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
	}

	:global(.dark) .form-group input:focus {
		border-color: #818cf8;
		box-shadow: 0 0 0 3px rgba(129, 140, 248, 0.2);
	}

	.form-group input:disabled {
		background-color: #f7fafc;
		cursor: not-allowed;
	}

	:global(.dark) .form-group input:disabled {
		background-color: #1f2937;
	}

	.form-group input::placeholder {
		color: #a0aec0;
	}

	:global(.dark) .form-group input::placeholder {
		color: #6b7280;
	}

	.company-select {
		width: 100%;
		padding: 0.75rem;
		border: 1px solid #cbd5e0;
		border-radius: 4px;
		font-size: 1rem;
		transition: border-color 0.15s ease-in-out;
		box-sizing: border-box;
		background-color: white;
		color: #2d3748;
		cursor: pointer;
	}

	:global(.dark) .company-select {
		background-color: #374151;
		border-color: #4b5563;
		color: #f3f4f6;
	}

	.company-select:focus {
		outline: none;
		border-color: #667eea;
		box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
	}

	:global(.dark) .company-select:focus {
		border-color: #818cf8;
		box-shadow: 0 0 0 3px rgba(129, 140, 248, 0.2);
	}

	.company-select:disabled {
		background-color: #f7fafc;
		cursor: not-allowed;
	}

	:global(.dark) .company-select:disabled {
		background-color: #1f2937;
	}

	.login-button {
		width: 100%;
		padding: 0.75rem;
		background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
		color: white;
		border: none;
		border-radius: 4px;
		font-size: 1rem;
		font-weight: 500;
		cursor: pointer;
		transition: transform 0.1s ease-in-out, box-shadow 0.15s ease-in-out;
	}

	:global(.dark) .login-button {
		background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
	}

	.login-button:hover:not(:disabled) {
		transform: translateY(-1px);
		box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
	}

	:global(.dark) .login-button:hover:not(:disabled) {
		box-shadow: 0 4px 12px rgba(99, 102, 241, 0.4);
	}

	.login-button:active:not(:disabled) {
		transform: translateY(0);
	}

	.login-button:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.secondary-button {
		width: 100%;
		padding: 0.75rem;
		background: #e2e8f0;
		color: #2d3748;
		border: none;
		border-radius: 4px;
		font-size: 0.875rem;
		font-weight: 500;
		cursor: pointer;
		margin-top: 0.75rem;
		transition: background-color 0.15s ease-in-out;
	}

	:global(.dark) .secondary-button {
		background: #374151;
		color: #e5e7eb;
	}

	.secondary-button:hover:not(:disabled) {
		background: #cbd5e0;
	}

	:global(.dark) .secondary-button:hover:not(:disabled) {
		background: #4b5563;
	}

	.secondary-button:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.setup-prompt {
		margin-top: 1.5rem;
		padding: 1rem;
		background-color: #f7fafc;
		border-radius: 4px;
		text-align: center;
	}

	:global(.dark) .setup-prompt {
		background-color: #374151;
	}

	.setup-prompt p {
		margin: 0 0 0.75rem 0;
		color: #4a5568;
		font-size: 0.875rem;
	}

	:global(.dark) .setup-prompt p {
		color: #9ca3af;
	}

	.company-row {
		display: flex;
		gap: 0.5rem;
	}

	.company-row .company-select {
		flex: 1;
	}

	.new-company-btn {
		padding: 0.75rem 1rem;
		background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
		color: white;
		border: none;
		border-radius: 4px;
		font-size: 1.25rem;
		font-weight: bold;
		cursor: pointer;
		transition: transform 0.1s ease-in-out, box-shadow 0.15s ease-in-out;
		line-height: 1;
	}

	:global(.dark) .new-company-btn {
		background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
	}

	.new-company-btn:hover:not(:disabled) {
		transform: translateY(-1px);
		box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
	}

	.new-company-btn:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.help-text {
		display: block;
		margin-top: 0.25rem;
		font-size: 0.75rem;
		color: #718096;
	}

	:global(.dark) .help-text {
		color: #9ca3af;
	}
</style>
