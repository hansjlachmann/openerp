<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { theme } from '$lib/stores/theme';
	import { currentUser } from '$lib/stores/user';
	import { session, currentCompany } from '$stores/session';
	import { api } from '$lib/services/api';
	import { t, MSG, MENU } from '$lib/services/i18n.svelte';

	let loading = $state(true);
	let currentTheme = $state<'light' | 'dark'>('light');
	let showUserMenu = $state(false);
	let showLanguageMenu = $state(false);
	let showCompanyMenu = $state(false);
	let languages = $state<{ code: string; name: string }[]>([]);
	let companies = $state<string[]>([]);
	let currentLanguage = $state('en-US');
	let currentCompanyName = $state('');
	let changingLanguage = $state(false);
	let changingCompany = $state(false);
	let version = $state('...');

	theme.subscribe((value) => {
		currentTheme = value;
	});

	// Subscribe to session for language and company updates
	session.subscribe((sess) => {
		if (sess.language) {
			currentLanguage = sess.language;
		}
		if (sess.company) {
			currentCompanyName = sess.company;
		}
	});

	// Track previous user ID to detect login/logout changes
	let prevUserID: string | null = null;

	// Reload companies when user changes (login/logout/switch)
	$effect(() => {
		const user = $currentUser;
		const userID = user?.user_id ?? null;
		if (userID !== prevUserID) {
			prevUserID = userID;
			if (userID) {
				// User just logged in - reload companies (filtered by access)
				api.getCompanies().then(comps => { companies = comps; }).catch(() => {});
			}
		}
	});

	onMount(async () => {
		try {
			// Load current user info from storage
			currentUser.loadFromStorage();
			// Load available languages, companies and version in parallel
			const [langs, comps, ver] = await Promise.all([
				api.getLanguages(),
				api.getCompanies(),
				api.getVersion()
			]);
			languages = langs;
			companies = comps;
			version = ver;
		} catch (err) {
			console.error('Error loading data:', err);
		} finally {
			loading = false;
		}
	});

	function toggleTheme() {
		theme.toggle();
	}

	async function handleLogout() {
		showUserMenu = false;
		try {
			await api.logout();
			currentUser.clear();
			goto('/login');
		} catch (err) {
			console.error('Logout error:', err);
			// Even if API call fails, remove local data and redirect
			currentUser.clear();
			goto('/login');
		}
	}

	function toggleUserMenu() {
		showUserMenu = !showUserMenu;
		showLanguageMenu = false;
		showCompanyMenu = false;
	}

	function toggleLanguageMenu() {
		showLanguageMenu = !showLanguageMenu;
		showCompanyMenu = false;
	}

	function toggleCompanyMenu() {
		showCompanyMenu = !showCompanyMenu;
		showLanguageMenu = false;
	}

	async function handleLanguageChange(langCode: string) {
		if (langCode === currentLanguage || changingLanguage) return;

		changingLanguage = true;
		try {
			await api.setLanguage(langCode, true);
			currentLanguage = langCode;
			session.setLanguage(langCode);
			showLanguageMenu = false;
			showUserMenu = false;

			// Reload the page to apply new translations
			window.location.reload();
		} catch (err) {
			console.error('Error changing language:', err);
		} finally {
			changingLanguage = false;
		}
	}

	async function handleCompanyChange(companyName: string) {
		if (companyName === currentCompanyName || changingCompany) return;

		changingCompany = true;
		try {
			await api.setCompany(companyName);
			currentCompanyName = companyName;
			session.setCompany(companyName);
			showCompanyMenu = false;
			showUserMenu = false;

			// Reload the page to apply company change
			window.location.reload();
		} catch (err) {
			console.error('Error changing company:', err);
		} finally {
			changingCompany = false;
		}
	}

	// Close dropdown when clicking outside
	function handleClickOutside(event: MouseEvent) {
		const target = event.target as HTMLElement;
		const userMenuButton = target.closest('[data-user-menu]');
		const userMenuDropdown = target.closest('[data-user-dropdown]');

		if (!userMenuButton && !userMenuDropdown && showUserMenu) {
			showUserMenu = false;
			showLanguageMenu = false;
			showCompanyMenu = false;
		}
	}

	onMount(() => {
		document.addEventListener('click', handleClickOutside);
		return () => {
			document.removeEventListener('click', handleClickOutside);
		};
	});

	// Get language display name
	function getLanguageName(code: string): string {
		const lang = languages.find((l) => l.code === code);
		return lang?.name || code;
	}

	// Get short language code for display (e.g., "EN" from "en-US")
	function getShortCode(code: string): string {
		return code.split('-')[0].toUpperCase();
	}
</script>

{#if loading}
	<div class="menu-bar bg-nav-blue text-white">
		<div class="px-4 py-2 text-sm">{t(MSG.LOADING)}</div>
	</div>
{:else}
	<nav class="menu-bar bg-nav-blue text-white">
		<div class="flex items-center gap-2 w-full">
			<!-- Home button -->
			<a
				href="/"
				class="flex items-center gap-2 px-4 py-2 hover:bg-white/10 rounded transition-colors font-semibold border-r border-white/20 mr-2"
				title={t(MENU.HOME)}
			>
				<svg
					xmlns="http://www.w3.org/2000/svg"
					class="h-5 w-5"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"
					/>
				</svg>
				<span>{t(MENU.HOME)}</span>
			</a>

			<!-- Session information -->
			<div class="flex items-center gap-4 text-sm text-white/80 border-r border-white/20 pr-4 mr-2">
				{#if $currentCompany}
					<span class="font-medium">{$currentCompany}</span>
				{/if}
				{#if $session.userName}
					<span>User: {$session.userName}</span>
				{/if}
				{#if $session.language}
					<span>Language: {$session.language}</span>
				{/if}
			</div>

			<!-- Spacer to push user menu to the right -->
			<div class="flex-1"></div>

			<!-- User menu -->
			{#if $currentUser}
				<div class="relative border-l border-white/20 pl-3 ml-2">
					<button
						data-user-menu
						onclick={toggleUserMenu}
						class="flex items-center gap-2 px-3 py-2 hover:bg-white/10 rounded transition-colors"
						title={t(MENU.USER_MENU)}
					>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							class="h-5 w-5"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"
							/>
						</svg>
						<span class="text-sm font-medium">{$currentUser.user_name || $currentUser.user_id}</span>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							class="h-4 w-4"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M19 9l-7 7-7-7"
							/>
						</svg>
					</button>

					<!-- User dropdown menu -->
					{#if showUserMenu}
						<div data-user-dropdown class="absolute right-0 mt-2 w-56 bg-white dark:bg-gray-800 rounded-md shadow-lg border border-gray-200 dark:border-gray-700 z-50">
							<div class="px-4 py-3 border-b border-gray-200 dark:border-gray-700">
								<p class="text-sm font-medium text-gray-900 dark:text-white">{$currentUser.user_name}</p>
								<p class="text-xs text-gray-500 dark:text-gray-400">{$currentUser.email || $currentUser.user_id}</p>
							</div>
							<div class="py-1">
								<!-- Language selector -->
								<div class="relative">
									<button
										onclick={toggleLanguageMenu}
										class="w-full text-left px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center justify-between"
									>
										<span class="flex items-center gap-2">
											<svg
												xmlns="http://www.w3.org/2000/svg"
												class="h-4 w-4"
												fill="none"
												viewBox="0 0 24 24"
												stroke="currentColor"
											>
												<path
													stroke-linecap="round"
													stroke-linejoin="round"
													stroke-width="2"
													d="M3 5h12M9 3v2m1.048 9.5A18.022 18.022 0 016.412 9m6.088 9h7M11 21l5-10 5 10M12.751 5C11.783 10.77 8.07 15.61 3 18.129"
												/>
											</svg>
											{t(MENU.LANGUAGE)}
										</span>
										<span class="flex items-center gap-1 text-xs text-gray-500 dark:text-gray-400">
											{getShortCode(currentLanguage)}
											<svg
												xmlns="http://www.w3.org/2000/svg"
												class="h-3 w-3"
												fill="none"
												viewBox="0 0 24 24"
												stroke="currentColor"
											>
												<path
													stroke-linecap="round"
													stroke-linejoin="round"
													stroke-width="2"
													d="M9 5l7 7-7 7"
												/>
											</svg>
										</span>
									</button>

									<!-- Language submenu -->
									{#if showLanguageMenu}
										<div class="absolute right-full top-0 mr-1 w-48 bg-white dark:bg-gray-800 rounded-md shadow-lg border border-gray-200 dark:border-gray-700">
											{#each languages as lang}
												<button
													onclick={() => handleLanguageChange(lang.code)}
													disabled={changingLanguage}
													class="w-full text-left px-4 py-2 text-sm hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center justify-between disabled:opacity-50"
													class:text-blue-600={lang.code === currentLanguage}
													class:dark:text-blue-400={lang.code === currentLanguage}
													class:font-medium={lang.code === currentLanguage}
													class:text-gray-700={lang.code !== currentLanguage}
													class:dark:text-gray-300={lang.code !== currentLanguage}
												>
													{lang.name}
													{#if lang.code === currentLanguage}
														<svg
															xmlns="http://www.w3.org/2000/svg"
															class="h-4 w-4"
															fill="none"
															viewBox="0 0 24 24"
															stroke="currentColor"
														>
															<path
																stroke-linecap="round"
																stroke-linejoin="round"
																stroke-width="2"
																d="M5 13l4 4L19 7"
															/>
														</svg>
													{/if}
												</button>
											{/each}
										</div>
									{/if}
								</div>

								<!-- Company selector -->
								{#if companies.length > 1}
									<div class="relative">
										<button
											onclick={toggleCompanyMenu}
											class="w-full text-left px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center justify-between"
										>
											<span class="flex items-center gap-2">
												<svg
													xmlns="http://www.w3.org/2000/svg"
													class="h-4 w-4"
													fill="none"
													viewBox="0 0 24 24"
													stroke="currentColor"
												>
													<path
														stroke-linecap="round"
														stroke-linejoin="round"
														stroke-width="2"
														d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4"
													/>
												</svg>
												{t(MENU.COMPANY)}
											</span>
											<span class="flex items-center gap-1 text-xs text-gray-500 dark:text-gray-400">
												{currentCompanyName || '—'}
												<svg
													xmlns="http://www.w3.org/2000/svg"
													class="h-3 w-3"
													fill="none"
													viewBox="0 0 24 24"
													stroke="currentColor"
												>
													<path
														stroke-linecap="round"
														stroke-linejoin="round"
														stroke-width="2"
														d="M9 5l7 7-7 7"
													/>
												</svg>
											</span>
										</button>

										<!-- Company submenu -->
										{#if showCompanyMenu}
											<div class="absolute right-full top-0 mr-1 w-48 bg-white dark:bg-gray-800 rounded-md shadow-lg border border-gray-200 dark:border-gray-700 max-h-64 overflow-y-auto">
												{#each companies as company}
													<button
														onclick={() => handleCompanyChange(company)}
														disabled={changingCompany}
														class="w-full text-left px-4 py-2 text-sm hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center justify-between disabled:opacity-50"
														class:text-blue-600={company === currentCompanyName}
														class:dark:text-blue-400={company === currentCompanyName}
														class:font-medium={company === currentCompanyName}
														class:text-gray-700={company !== currentCompanyName}
														class:dark:text-gray-300={company !== currentCompanyName}
													>
														{company}
														{#if company === currentCompanyName}
															<svg
																xmlns="http://www.w3.org/2000/svg"
																class="h-4 w-4"
																fill="none"
																viewBox="0 0 24 24"
																stroke="currentColor"
															>
																<path
																	stroke-linecap="round"
																	stroke-linejoin="round"
																	stroke-width="2"
																	d="M5 13l4 4L19 7"
																/>
															</svg>
														{/if}
													</button>
												{/each}
											</div>
										{/if}
									</div>
								{/if}

								<!-- Divider -->
								<div class="border-t border-gray-200 dark:border-gray-700 my-1"></div>

								<!-- Sign out -->
								<button
									onclick={handleLogout}
									class="w-full text-left px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
								>
									<svg
										xmlns="http://www.w3.org/2000/svg"
										class="h-4 w-4"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											stroke-width="2"
											d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"
										/>
									</svg>
									{t(MENU.SIGN_OUT)}
								</button>
							</div>
						</div>
					{/if}
				</div>
			{/if}

			<!-- Theme toggle button -->
			<button
				onclick={toggleTheme}
				class="flex items-center gap-2 px-3 py-2 hover:bg-white/10 rounded transition-colors"
				title={currentTheme === 'light' ? 'Switch to dark mode' : 'Switch to light mode'}
				aria-label="Toggle theme"
			>
				{#if currentTheme === 'light'}
					<!-- Moon icon for dark mode -->
					<svg
						xmlns="http://www.w3.org/2000/svg"
						class="h-5 w-5"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"
						/>
					</svg>
				{:else}
					<!-- Sun icon for light mode -->
					<svg
						xmlns="http://www.w3.org/2000/svg"
						class="h-5 w-5"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"
						/>
					</svg>
				{/if}
			</button>

			<!-- Version -->
			<span class="text-xs text-white/70 pl-2 border-l border-white/20">v{version}</span>
		</div>
	</nav>
{/if}

<style>
	.menu-bar {
		@apply flex items-center px-4 py-2 shadow-md;
		min-height: 3rem;
	}
</style>
