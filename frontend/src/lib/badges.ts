export type BadgeInsurance = '' | '6w' | '120w' | 'lti';

export interface BadgeDisplay {
	tier: number;
	title: string;
	symbol: string;
}

const TIER_SYMBOLS: Record<number, string> = {
	1: '▲',
	2: '●',
	3: '◆',
	4: '◈',
	5: '★'
};

const TIER_LABELS: Record<number, string> = {
	1: 'Common',
	2: 'Uncommon',
	3: 'Rare',
	4: 'Epic',
	5: 'Legendary'
};

export const BADGE_DISPLAY_BY_KEY: Record<string, BadgeDisplay> = {
	first_blood: { tier: 1, title: 'First Blood', symbol: '▲' },
	quick_shot: { tier: 1, title: 'Quick Shot', symbol: '▲' },
	market_founder: { tier: 1, title: 'Market Founder', symbol: '▲' },
	eternal_optimist: { tier: 2, title: 'Eternal Optimist', symbol: '●' },
	doomsayer: { tier: 2, title: 'Doomsayer', symbol: '●' },
	market_maven: { tier: 2, title: 'Market Maven', symbol: '●' },
	seasoned_trader: { tier: 2, title: 'Seasoned Trader', symbol: '●' },
	skeptic: { tier: 2, title: 'Skeptic', symbol: '●' },
	portfolio_manager: { tier: 2, title: 'Portfolio Manager', symbol: '●' },
	serial_founder: { tier: 2, title: 'Serial Founder', symbol: '●' },
	bug_prophet: { tier: 3, title: 'Bug Prophet', symbol: '◆' },
	market_obsessed: { tier: 3, title: 'Market Obsessed', symbol: '◆' },
	universe_citizen: { tier: 3, title: 'Universe Citizen', symbol: '◆' },
	galaxy_brained: { tier: 4, title: 'Galaxy Brained', symbol: '◈' },
	oracle: { tier: 4, title: 'Oracle', symbol: '◈' },
	citizen_backer: { tier: 1, title: 'Citizen Backer', symbol: '▲' },
	professional_bug_finder: { tier: 1, title: 'Professional Bug Finder', symbol: '▲' },
	aurora_pilot: { tier: 1, title: 'Aurora Pilot', symbol: '▲' },
	roadmap_reader: { tier: 1, title: 'Roadmap Reader', symbol: '▲' },
	warp_speed: { tier: 1, title: 'Warp Speed', symbol: '▲' },
	mostly_backer: { tier: 2, title: 'Mostly Backer', symbol: '●' },
	hangar_queen: { tier: 2, title: 'Hangar Queen', symbol: '●' },
	tech_preview_survivor: { tier: 2, title: 'Tech Preview Survivor', symbol: '●' },
	star_gazer: { tier: 2, title: 'Star Gazer', symbol: '●' },
	alpha_tester: { tier: 2, title: 'Alpha Tester', symbol: '●' },
	space_whale: { tier: 2, title: 'Space Whale', symbol: '●' },
	bugged_not_broken: { tier: 2, title: 'Bugged, Not Broken', symbol: '●' },
	verse_veteran: { tier: 2, title: "'Verse Veteran", symbol: '●' },
	alpha_optimist: { tier: 2, title: 'Alpha Optimist', symbol: '●' },
	q4_enjoyer: { tier: 2, title: 'Q4 Enjoyer', symbol: '●' },
	persistent_citizen: { tier: 3, title: 'Persistent Universe Citizen', symbol: '◆' },
	org_leader: { tier: 3, title: 'Org Leader', symbol: '◆' },
	'900i_enjoyer': { tier: 3, title: '900i Enjoyer', symbol: '◆' },
	system_colonist: { tier: 3, title: 'System Colonist', symbol: '◆' },
	citizencon_pilgrim: { tier: 3, title: 'CitizenCon Pilgrim', symbol: '◆' },
	idris_captain: { tier: 4, title: 'Idris Captain', symbol: '◈' },
	backer_royalty: { tier: 4, title: 'Backer Royalty', symbol: '◈' },
	fleet_commander_badge: { tier: 4, title: 'Fleet Commander', symbol: '◈' },
	golden_ticket: { tier: 5, title: 'Golden Ticket', symbol: '★' },
	unobtainium: { tier: 5, title: 'Unobtainium Tier', symbol: '★' },
	ensign: { tier: 1, title: 'Ensign', symbol: '⚓' },
	lieutenant: { tier: 2, title: 'Lieutenant', symbol: '⚔' },
	commander: { tier: 3, title: 'Commander', symbol: '🛡' },
	captain: { tier: 4, title: 'Captain', symbol: '👑' },
	coin_admiral: { tier: 5, title: 'Coin Admiral', symbol: '🌟' }
};

export function getBadgeTierSymbol(tier: number): string {
	return TIER_SYMBOLS[tier] ?? TIER_SYMBOLS[1];
}

export function getBadgeTierLabel(tier: number): string {
	return TIER_LABELS[tier] ?? TIER_LABELS[1];
}

export function getBadgeDisplay(key: string | null | undefined): BadgeDisplay | undefined {
	if (!key) return undefined;
	return BADGE_DISPLAY_BY_KEY[key];
}

export function formatBadgeInsurance(
	insurance: string | null | undefined,
	style: 'short' | 'long' = 'short'
): string {
	if (!insurance) return '';
	if (insurance === 'lti') return 'LTI';
	if (insurance === '120w') return style === 'long' ? '120 Weeks' : '120W Ins.';
	if (insurance === '6w') return style === 'long' ? '6 Weeks' : '6W Ins.';
	return insurance;
}