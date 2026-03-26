import type { MarketCategory } from './types';

/** Human-readable labels for each market category. */
export const CATEGORY_LABELS: Record<MarketCategory, string> = {
	bug_fixes: 'Bug Fix',
	feature_delivery: 'Feature / Patch',
	patch_timing: 'Patch Timing',
	community_events: 'Community Event',
	meta: 'Meta'
};

/** Ordered list of categories for filter dropdowns (includes an "all" sentinel). */
export const CATEGORY_FILTER_OPTIONS: { value: MarketCategory | ''; label: string }[] = [
	{ value: '', label: 'All Categories' },
	{ value: 'bug_fixes', label: 'Bug Fix' },
	{ value: 'feature_delivery', label: 'Feature / Patch' },
	{ value: 'patch_timing', label: 'Patch Timing' },
	{ value: 'community_events', label: 'Community Event' },
	{ value: 'meta', label: 'Meta' }
];

/** Ordered list of categories for market creation forms (no "all" sentinel). */
export const CATEGORY_CREATE_OPTIONS: { value: MarketCategory; label: string }[] = [
	{ value: 'bug_fixes', label: 'Bug Fix' },
	{ value: 'feature_delivery', label: 'Feature / Patch' },
	{ value: 'patch_timing', label: 'Patch Timing' },
	{ value: 'community_events', label: 'Community Event' },
	{ value: 'meta', label: 'Meta' }
];
