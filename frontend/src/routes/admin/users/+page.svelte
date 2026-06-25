<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import Mail from '@lucide/svelte/icons/mail';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import Copy from '@lucide/svelte/icons/copy';
	import Check from '@lucide/svelte/icons/check';
	import { i18n } from '$lib/i18n/state.svelte';
	import { auth } from '$lib/admin/auth.svelte';
	import {
		deleteUser,
		inviteUser,
		listInvites,
		listUsers,
		revokeInvite,
		type AdminInvite
	} from '$lib/admin/api';
	import type { AdminUser } from '$lib/admin/auth.svelte';
	import PageHeader from '$lib/admin/components/PageHeader.svelte';
	import Modal from '$lib/admin/components/Modal.svelte';

	let users: AdminUser[] = $state([]);
	let invites: AdminInvite[] = $state([]);
	let loading = $state(true);

	let inviteOpen = $state(false);
	let inviteEmail = $state('');
	let inviting = $state(false);
	let inviteError = $state<string | null>(null);
	let lastInvite: AdminInvite | null = $state(null);
	let copied = $state(false);

	async function refresh() {
		users = await listUsers();
		invites = await listInvites();
	}

	onMount(async () => {
		await refresh();
		loading = false;
	});

	function openInvite() {
		inviteOpen = true;
		inviteEmail = '';
		inviteError = null;
		lastInvite = null;
		copied = false;
	}

	function closeInvite() {
		inviteOpen = false;
	}

	async function onInvite(e: SubmitEvent) {
		e.preventDefault();
		inviting = true;
		inviteError = null;
		try {
			lastInvite = await inviteUser(inviteEmail);
			inviteEmail = '';
			await refresh();
		} catch (err) {
			inviteError = err instanceof Error ? err.message : String(err);
		} finally {
			inviting = false;
		}
	}

	function inviteLink(token: string): string {
		const base =
			typeof window !== 'undefined' ? `${window.location.origin}` : 'https://running.rydback.net';
		return `${base}${resolve('/admin/accept')}?token=${token}`;
	}

	async function copyLink(token: string) {
		try {
			await navigator.clipboard.writeText(inviteLink(token));
			copied = true;
			setTimeout(() => (copied = false), 1500);
		} catch {
			/* ignore */
		}
	}

	async function onRevoke(invite: AdminInvite) {
		if (!confirm(i18n.m.admin.common.confirmDelete)) return;
		await revokeInvite(invite.id);
		await refresh();
	}

	async function onDeleteUser(user: AdminUser) {
		if (!confirm(i18n.m.admin.common.confirmDelete)) return;
		try {
			await deleteUser(user.id);
			await refresh();
		} catch (err) {
			alert(err instanceof Error ? err.message : String(err));
		}
	}

	const pendingInvites = $derived(invites.filter((i) => !i.acceptedAt));
</script>

<section class="space-y-6">
	<PageHeader title={i18n.m.admin.users.heading}>
		{#snippet action()}
			<button
				type="button"
				class="btn preset-filled-primary-500 inline-flex items-center gap-2"
				onclick={openInvite}
			>
				<Mail class="size-4" />
				{i18n.m.admin.users.invite}
			</button>
		{/snippet}
	</PageHeader>

	{#if loading}
		<p class="opacity-70 text-sm">{i18n.m.admin.common.loading}</p>
	{:else}
		<section class="space-y-3">
			<h2 class="text-sm uppercase tracking-wide opacity-70">
				{i18n.m.admin.users.usersHeading}
			</h2>
			<div class="card preset-filled-surface-50-950 border border-surface-200-800 overflow-hidden">
				<div class="table-wrap">
					<table class="table">
						<thead>
							<tr>
								<th>{i18n.m.admin.users.columnEmail}</th>
								<th>{i18n.m.admin.users.columnCreated}</th>
								<th class="w-28 text-right">{i18n.m.admin.common.actions}</th>
							</tr>
						</thead>
						<tbody>
							{#each users as user (user.id)}
								<tr>
									<td>
										<span class="font-medium">{user.email}</span>
										{#if auth.user?.id === user.id}
											<span
												class="ml-2 inline-block rounded-full bg-primary-100 dark:bg-primary-900/40 text-primary-700 dark:text-primary-200 px-2 py-0.5 text-xs"
											>
												{i18n.m.admin.users.youBadge}
											</span>
										{/if}
									</td>
									<td class="text-sm opacity-80">
										{new Date(user.createdAt).toLocaleDateString()}
									</td>
									<td class="text-right">
										{#if auth.user?.id !== user.id}
											<button
												type="button"
												class="btn-icon btn-icon-sm hover:bg-error-100 hover:text-error-700 dark:hover:bg-error-900/40 dark:hover:text-error-200"
												aria-label={i18n.m.admin.common.delete}
												title={i18n.m.admin.common.delete}
												onclick={() => onDeleteUser(user)}
											>
												<Trash2 class="size-4" />
											</button>
										{/if}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		</section>

		{#if pendingInvites.length > 0}
			<section class="space-y-3">
				<h2 class="text-sm uppercase tracking-wide opacity-70">
					{i18n.m.admin.users.pendingHeading}
				</h2>
				<div
					class="card preset-filled-surface-50-950 border border-surface-200-800 overflow-hidden"
				>
					<div class="table-wrap">
						<table class="table">
							<thead>
								<tr>
									<th>{i18n.m.admin.users.columnEmail}</th>
									<th>{i18n.m.admin.users.columnInvitedBy}</th>
									<th>{i18n.m.admin.users.columnCreated}</th>
									<th class="w-40 text-right">{i18n.m.admin.common.actions}</th>
								</tr>
							</thead>
							<tbody>
								{#each pendingInvites as invite (invite.id)}
									<tr>
										<td>
											<span class="font-medium">{invite.email}</span>
											<span
												class="ml-2 inline-block rounded-full bg-warning-100 dark:bg-warning-900/40 text-warning-800 dark:text-warning-200 px-2 py-0.5 text-xs"
											>
												{i18n.m.admin.users.pendingBadge}
											</span>
										</td>
										<td class="text-sm opacity-80">{invite.createdByEmail}</td>
										<td class="text-sm opacity-80">
											{new Date(invite.createdAt).toLocaleDateString()}
										</td>
										<td class="text-right space-x-1">
											<button
												type="button"
												class="btn-icon btn-icon-sm hover:bg-surface-100-900"
												aria-label="Copy link"
												title={inviteLink(invite.token)}
												onclick={() => copyLink(invite.token)}
											>
												<Copy class="size-4" />
											</button>
											<button
												type="button"
												class="btn-icon btn-icon-sm hover:bg-error-100 hover:text-error-700 dark:hover:bg-error-900/40 dark:hover:text-error-200"
												aria-label={i18n.m.admin.users.revoke}
												title={i18n.m.admin.users.revoke}
												onclick={() => onRevoke(invite)}
											>
												<Trash2 class="size-4" />
											</button>
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				</div>
			</section>
		{/if}
	{/if}
</section>

<Modal open={inviteOpen} title={i18n.m.admin.users.inviteHeading} onClose={closeInvite}>
	<form class="space-y-4" onsubmit={onInvite}>
		<label class="block space-y-1">
			<span class="text-sm font-medium">{i18n.m.admin.users.inviteEmailLabel}</span>
			<input type="email" required bind:value={inviteEmail} class="input" />
		</label>

		{#if inviteError}
			<p class="text-sm text-error-600 dark:text-error-300">{inviteError}</p>
		{/if}

		{#if lastInvite}
			<div
				class="rounded-lg border border-success-200 dark:border-success-800 bg-success-50 dark:bg-success-900/30 px-3 py-2 space-y-2 text-sm"
			>
				<p>{i18n.m.admin.users.inviteCreated}</p>
				<div class="flex items-center gap-2">
					<code class="flex-1 text-xs break-all font-mono opacity-90">
						{inviteLink(lastInvite.token)}
					</code>
					<button
						type="button"
						class="btn-icon btn-icon-sm hover:bg-surface-100-900"
						aria-label="Copy"
						onclick={() => lastInvite && copyLink(lastInvite.token)}
					>
						{#if copied}
							<Check class="size-4 text-success-600" />
						{:else}
							<Copy class="size-4" />
						{/if}
					</button>
				</div>
			</div>
		{/if}

		<div class="flex items-center justify-end gap-2 pt-2">
			<button type="button" class="btn preset-tonal-surface" onclick={closeInvite}>
				{i18n.m.admin.common.cancel}
			</button>
			<button type="submit" disabled={inviting} class="btn preset-filled-primary-500">
				{inviting ? i18n.m.admin.common.saving : i18n.m.admin.users.inviteSubmit}
			</button>
		</div>
	</form>
</Modal>
