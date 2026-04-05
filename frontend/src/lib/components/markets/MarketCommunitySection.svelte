<script lang="ts">
	import { currentUser, isLoggedIn } from '$lib/stores/auth';
	import { getBadgeDisplay } from '$lib/badges';
	import { renderMarkdown } from '$lib/markdown';
	import type { Comment, MarketStatus } from '$lib/types';

	let {
		hasPosition,
		marketStatus,
		comments,
		requestingResolution,
		resolutionRequestMsg,
		resolutionLink = $bindable(),
		resolutionNote = $bindable(),
		showReportForm = $bindable(),
		reportReason = $bindable(),
		submittingReport,
		reportMsg,
		commentInput = $bindable(),
		postingComment,
		commentError,
		onRequestResolution,
		onSubmitReport,
		onPostComment,
		onDeleteComment
	}: {
		hasPosition: boolean;
		marketStatus: MarketStatus;
		comments: Comment[];
		requestingResolution: boolean;
		resolutionRequestMsg: string;
		resolutionLink: string;
		resolutionNote: string;
		showReportForm: boolean;
		reportReason: string;
		submittingReport: boolean;
		reportMsg: string;
		commentInput: string;
		postingComment: boolean;
		commentError: string;
		onRequestResolution: () => Promise<void>;
		onSubmitReport: () => Promise<void>;
		onPostComment: () => Promise<void>;
		onDeleteComment: (commentId: number) => Promise<void>;
	} = $props();

	function getRequiredBadgeDisplay(key: string | null | undefined) {
		return getBadgeDisplay(key)!;
	}
</script>

{#if hasPosition && marketStatus === 'active'}
	<div class="sc-card p-4 border border-amber-200 bg-amber-50/30">
		<h3 class="text-xs font-bold text-surface-500 uppercase tracking-[0.12em] mb-2">Request Resolution</h3>
		<p class="text-surface-600 text-xs mb-3">Think the resolution criteria has been met? Ask the mod team to review this market.</p>
		<label class="block mb-2" for="res-link">
			<span class="text-surface-600 text-xs uppercase tracking-wider font-semibold">Evidence Link <span class="normal-case font-normal text-surface-400">(optional)</span></span>
			<input
				id="res-link"
				type="url"
				bind:value={resolutionLink}
				placeholder="https://…"
				class="sc-input mt-1 text-sm"
			/>
		</label>
		<label class="block mb-3" for="res-note">
			<span class="text-surface-600 text-xs uppercase tracking-wider font-semibold">Note to mods <span class="normal-case font-normal text-surface-400">(optional)</span></span>
			<textarea
				id="res-note"
				bind:value={resolutionNote}
				rows="2"
				maxlength="500"
				placeholder="Brief explanation of why this should resolve…"
				class="sc-input mt-1 text-sm resize-none"
			></textarea>
		</label>
		<button
			onclick={onRequestResolution}
			disabled={requestingResolution}
			class="btn btn-sm border border-amber-500 text-amber-700 hover:bg-amber-100 text-xs uppercase tracking-wider transition-colors"
		>
			{requestingResolution ? 'Requesting…' : 'Request Resolution'}
		</button>
		{#if resolutionRequestMsg}
			<p class="text-xs mt-2 {resolutionRequestMsg.startsWith('Resolution request') ? 'text-green-700' : 'text-red-600'}">{resolutionRequestMsg}</p>
		{/if}
	</div>
{:else if hasPosition && marketStatus === 'resolution_requested'}
	<div class="sc-card p-4 border border-amber-200 bg-amber-50/30">
		<p class="text-amber-700 text-xs font-semibold">Resolution has been requested — mods will review shortly.</p>
	</div>
{/if}

{#if $isLoggedIn && (marketStatus === 'active' || marketStatus === 'resolution_requested')}
	<div class="mt-2">
		{#if !showReportForm}
			<button
				onclick={() => { showReportForm = true; }}
				class="text-surface-400 hover:text-red-500 text-xs transition-colors"
			>
				⚑ Report this market
			</button>
		{:else}
			<div class="sc-card p-4 border border-red-100">
				<h3 class="text-xs font-bold text-surface-500 uppercase tracking-[0.12em] mb-2">Report Market</h3>
				<label class="block mb-2" for="report-reason">
					<span class="text-surface-600 text-xs uppercase tracking-wider font-semibold">Reason</span>
					<textarea
						id="report-reason"
						bind:value={reportReason}
						rows="2"
						maxlength="500"
						placeholder="Why should mods review this market?"
						class="sc-input mt-1 text-sm resize-none"
					></textarea>
				</label>
				<div class="flex gap-2">
					<button
						onclick={onSubmitReport}
						disabled={submittingReport || reportReason.trim().length < 5}
						class="btn btn-sm bg-red-600 hover:bg-red-700 text-white text-xs uppercase tracking-wider disabled:opacity-50"
					>
						{submittingReport ? 'Submitting…' : 'Submit Report'}
					</button>
					<button
						onclick={() => { showReportForm = false; reportReason = ''; reportMsg = ''; }}
						class="btn btn-sm border border-surface-300 text-surface-500 text-xs"
					>
						Cancel
					</button>
				</div>
				{#if reportMsg}
					<p class="text-xs mt-2 {reportMsg.startsWith('Report submitted') ? 'text-green-700' : 'text-red-600'}">{reportMsg}</p>
				{/if}
			</div>
		{/if}
	</div>
{/if}

<div class="sc-card p-5">
	<h3 class="text-xs font-bold text-surface-500 uppercase tracking-[0.12em] mb-4">
		Discussion ({comments.length})
	</h3>

	{#if comments.length === 0}
		<p class="text-surface-400 text-xs text-center py-3">No comments yet — start the conversation.</p>
	{:else}
		<div class="space-y-5 mb-5">
			{#each comments as comment}
				<div>
					<div class="flex items-center gap-1.5 mb-1 flex-wrap">
						<span class="font-semibold text-surface-800 text-xs">{comment.author_name}</span>
						{#if getBadgeDisplay(comment.author_top_badge)}
							{@const badge = getRequiredBadgeDisplay(comment.author_top_badge)}
							<span class="comment-badge tier-{badge.tier}" title={badge.title}>{badge.symbol} {badge.title}</span>
						{/if}
						<span class="text-surface-400 text-[10px]">{new Date(comment.created_at).toLocaleString()}</span>
						{#if $currentUser?.is_moderator}
							<button
								onclick={() => onDeleteComment(comment.id)}
								class="text-[10px] text-red-400 hover:text-red-600 transition-colors ml-1"
							>Delete</button>
						{/if}
					</div>
					{#if comment.hidden}
						<div class="text-xs text-amber-700 bg-amber-50 border border-amber-200 rounded p-2 mb-1.5">
							<span class="font-bold">⚠ Under review</span> — your comment is only visible to you until a moderator clears it.
						</div>
						<div class="text-surface-500 text-sm prose prose-sm max-w-none opacity-60">
							{@html renderMarkdown(comment.content)}
						</div>
					{:else}
						<div class="text-surface-700 text-sm prose prose-sm max-w-none">
							{@html renderMarkdown(comment.content)}
						</div>
					{/if}
				</div>
			{/each}
		</div>
	{/if}

	{#if $isLoggedIn}
		<div class="{comments.length > 0 ? 'border-t border-surface-100 pt-4' : ''}">
			<textarea
				bind:value={commentInput}
				rows="3"
				maxlength="1000"
				placeholder="Add a comment… Markdown supported. Be excellent to each other."
				class="sc-input text-sm resize-none w-full"
			></textarea>
			<div class="flex items-center justify-between mt-2">
				<span class="text-[10px] text-surface-400">{commentInput.length}/1000 · auto-checked for abuse</span>
				<button
					onclick={onPostComment}
					disabled={postingComment || commentInput.trim().length === 0}
					class="btn btn-sm preset-filled-primary-500 text-xs uppercase tracking-wider disabled:opacity-50"
				>{postingComment ? 'Posting…' : 'Post'}</button>
			</div>
			{#if commentError}
				<p class="text-xs mt-2 {commentError.startsWith('⚠') ? 'text-amber-700' : 'text-red-600'}">{commentError}</p>
			{/if}
		</div>
	{:else}
		<p class="text-surface-400 text-xs text-center pt-3 border-t border-surface-100">
			<a href="/auth/login" class="text-primary-600 hover:underline">Log in</a> to join the discussion.
		</p>
	{/if}
</div>

<style>
	.comment-badge {
		display: inline-flex;
		align-items: center;
		gap: 0.2rem;
		padding: 0.1rem 0.45rem;
		border-radius: 9999px;
		font-size: 0.6rem;
		font-weight: 700;
		letter-spacing: 0.04em;
		white-space: nowrap;
	}
	.comment-badge.tier-1 {
		background: #f5ede0;
		border: 1px solid #d4b896;
		color: #7a5030;
	}
	.comment-badge.tier-2 {
		background: linear-gradient(90deg, #fef3d0, #fde68a);
		border: 1px solid #e6c96b;
		color: #92400e;
	}
	.comment-badge.tier-3 {
		background: linear-gradient(90deg, #fef9e0, #fef3c0);
		border: 1px solid #f0c040;
		color: #6b2d06;
	}
	.comment-badge.tier-4 {
		background: linear-gradient(90deg, #1c1008, #2d1c00);
		border: 1px solid #fbbf24;
		color: #fef3c7;
	}
	.comment-badge.tier-5 {
		background: linear-gradient(90deg, #0d0d0d, #1a1200);
		border: 1px solid #ffd700;
		color: #ffd700;
	}
</style>