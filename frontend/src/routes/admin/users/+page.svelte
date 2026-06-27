<script lang="ts">
	import { onMount } from 'svelte';
	import UserPlus from '@lucide/svelte/icons/user-plus';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import { i18n } from '$lib/i18n/state.svelte';
	import {
		adminCreateInvitation,
		adminListAdmins,
		adminListInvitations,
		adminRevokeInvitation
	} from '$lib/admin/api';
	import type { AdminAccount, AdminInvitation } from '$lib/types';
	import PageHeader from '$lib/admin/components/PageHeader.svelte';
	import Modal from '$lib/admin/components/Modal.svelte';

	let admins = $state<AdminAccount[]>([]);
	let invitations = $state<AdminInvitation[]>([]);
	let loading = $state(true);
	let listError = $state<string | null>(null);

	let inviting = $state(false);
	let inviteEmail = $state('');
	let inviteLocale = $state<'sv' | 'en'>('sv');
	let sending = $state(false);
	let formError = $state<string | null>(null);

	onMount(() => {
		void reload();
	});

	async function reload() {
		loading = true;
		listError = null;
		try {
			[admins, invitations] = await Promise.all([adminListAdmins(), adminListInvitations()]);
		} catch (err) {
			listError = err instanceof Error ? err.message : String(err);
		} finally {
			loading = false;
		}
	}

	function openInvite() {
		inviting = true;
		inviteEmail = '';
		inviteLocale = 'sv';
		formError = null;
	}

	function closeInvite() {
		inviting = false;
	}

	async function onSendInvite(ev: SubmitEvent) {
		ev.preventDefault();
		sending = true;
		formError = null;
		try {
			await adminCreateInvitation({ email: inviteEmail.trim(), locale: inviteLocale });
			closeInvite();
			await reload();
		} catch (err) {
			formError = err instanceof Error ? err.message : String(err);
		} finally {
			sending = false;
		}
	}

	async function onRevoke(inv: AdminInvitation) {
		if (!confirm(i18n.m.admin.users.confirmRevoke(inv.email))) return;
		try {
			await adminRevokeInvitation(inv.tokenId);
			await reload();
		} catch (err) {
			listError = err instanceof Error ? err.message : String(err);
		}
	}

	function statusBadgeClass(status: AdminInvitation['status']): string {
		switch (status) {
			case 'pending':
				return 'badge preset-filled-warning-500';
			case 'expired':
				return 'badge preset-tonal-surface';
			case 'used':
				return 'badge preset-filled-success-500';
			default:
				return 'badge preset-tonal-surface';
		}
	}

	function fmt(ts: string | undefined): string {
		if (!ts) return i18n.m.admin.users.never;
		try {
			return new Date(ts).toLocaleString();
		} catch {
			return ts;
		}
	}
</script>

<section class="space-y-6">
	<PageHeader
		title={i18n.m.admin.users.heading}
		description={i18n.m.admin.users.description}
	>
		{#snippet action()}
			<button
				type="button"
				class="btn preset-filled-primary-500 inline-flex items-center gap-2"
				onclick={openInvite}
			>
				<UserPlus class="size-4" />
				{i18n.m.admin.users.invite}
			</button>
		{/snippet}
	</PageHeader>

	{#if listError}
		<p class="text-sm text-error-600 dark:text-error-300">{listError}</p>
	{/if}

	{#if loading}
		<p class="opacity-70 text-sm">{i18n.m.admin.common.loading}</p>
	{:else}
		<div class="space-y-2">
			<h2 class="text-sm font-semibold uppercase tracking-wide opacity-70">
				{i18n.m.admin.users.currentAdminsHeading}
			</h2>
			<div
				class="card preset-filled-surface-50-950 border border-surface-200-800 overflow-hidden"
			>
				<div class="table-wrap">
					<table class="table">
						<thead>
							<tr>
								<th>{i18n.m.admin.users.columnEmail}</th>
								<th>{i18n.m.admin.users.columnLocale}</th>
								<th>{i18n.m.admin.users.columnCreatedAt}</th>
								<th>{i18n.m.admin.users.columnLastLogin}</th>
							</tr>
						</thead>
						<tbody>
							{#each admins as a (a.id)}
								<tr>
									<td class="font-medium text-sm">{a.email}</td>
									<td class="text-sm uppercase">{a.locale || 'sv'}</td>
									<td class="text-sm">{fmt(a.createdAt)}</td>
									<td class="text-sm">{fmt(a.lastLoginAt)}</td>
								</tr>
							{/each}
							{#if admins.length === 0}
								<tr>
									<td colspan="4" class="opacity-70 text-sm py-3">
										{i18n.m.admin.common.noneYet}
									</td>
								</tr>
							{/if}
						</tbody>
					</table>
				</div>
			</div>
		</div>

		<div class="space-y-2">
			<h2 class="text-sm font-semibold uppercase tracking-wide opacity-70">
				{i18n.m.admin.users.pendingInvitesHeading}
			</h2>
			{#if invitations.length === 0}
				<p class="opacity-70 text-sm">{i18n.m.admin.users.noPendingInvites}</p>
			{:else}
				<div
					class="card preset-filled-surface-50-950 border border-surface-200-800 overflow-hidden"
				>
					<div class="table-wrap">
						<table class="table">
							<thead>
								<tr>
									<th>{i18n.m.admin.users.columnEmail}</th>
									<th>{i18n.m.admin.users.columnLocale}</th>
									<th>{i18n.m.admin.users.columnInvitedBy}</th>
									<th>{i18n.m.admin.users.columnStatus}</th>
									<th>{i18n.m.admin.users.columnCreatedAt}</th>
									<th>{i18n.m.admin.users.columnExpiresAt}</th>
									<th class="text-right">{i18n.m.admin.common.actions}</th>
								</tr>
							</thead>
							<tbody>
								{#each invitations as inv (inv.tokenId)}
									<tr>
										<td class="font-medium text-sm">{inv.email}</td>
										<td class="text-sm uppercase">{inv.locale || 'sv'}</td>
										<td class="text-sm">{inv.invitedByMail || '—'}</td>
										<td>
											<span class={statusBadgeClass(inv.status)}>
												{i18n.m.admin.users.statusLabel(inv.status)}
											</span>
										</td>
										<td class="text-sm">{fmt(inv.createdAt)}</td>
										<td class="text-sm">{fmt(inv.expiresAt)}</td>
										<td class="text-right">
											{#if inv.status === 'pending'}
												<button
													type="button"
													class="btn-icon btn-icon-sm hover:bg-error-100 hover:text-error-700 dark:hover:bg-error-900/40 dark:hover:text-error-200"
													aria-label={i18n.m.admin.users.revoke}
													title={i18n.m.admin.users.revoke}
													onclick={() => onRevoke(inv)}
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
			{/if}
		</div>
	{/if}
</section>

<Modal open={inviting} title={i18n.m.admin.users.inviteHeading} onClose={closeInvite}>
	<form class="space-y-4" onsubmit={onSendInvite}>
		<label class="block space-y-1">
			<span class="text-sm font-medium">{i18n.m.admin.users.emailLabel}</span>
			<input
				type="email"
				autocomplete="email"
				required
				bind:value={inviteEmail}
				class="input"
			/>
		</label>
		<label class="block space-y-1">
			<span class="text-sm font-medium">{i18n.m.admin.users.localeLabel}</span>
			<select bind:value={inviteLocale} class="select">
				<option value="sv">{i18n.m.admin.users.localeSv}</option>
				<option value="en">{i18n.m.admin.users.localeEn}</option>
			</select>
		</label>

		{#if formError}
			<p class="text-sm text-error-600 dark:text-error-300">{formError}</p>
		{/if}

		<div class="flex items-center justify-end gap-2 pt-2">
			<button type="button" class="btn preset-tonal-surface" onclick={closeInvite}>
				{i18n.m.admin.common.cancel}
			</button>
			<button type="submit" disabled={sending} class="btn preset-filled-primary-500">
				{sending ? i18n.m.admin.users.sending : i18n.m.admin.users.sendInvite}
			</button>
		</div>
	</form>
</Modal>
