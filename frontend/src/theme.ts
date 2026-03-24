import type { CustomThemeConfig } from '@skeletonlabs/tw-plugin';

/**
 * Origin Jumpworks Luxury Gold theme for ScolyMarket.
 * Inspired by the premium Origin yacht aesthetic — warm gold on deep charcoal.
 */
export const luxuryGoldTheme: CustomThemeConfig = {
	name: 'luxury-gold',
	properties: {
		// === Typefaces ===
		'--theme-font-family-base': `'Inter', 'system-ui', sans-serif`,
		'--theme-font-family-heading': `'Inter', 'system-ui', sans-serif`,
		'--theme-font-color-base': '255 255 255',
		'--theme-font-color-dark': '15 17 23',
		'--theme-rounded-base': '6px',
		'--theme-rounded-container': '8px',
		'--theme-border-base': '1px',

		// === Primary: Warm Gold (#D4A843) ===
		'--color-primary-50': '249 241 223',
		'--color-primary-100': '245 232 195',
		'--color-primary-200': '237 214 140',
		'--color-primary-300': '228 196 101',
		'--color-primary-400': '212 168 67',
		'--color-primary-500': '212 168 67',   // #D4A843 base
		'--color-primary-600': '190 148 52',
		'--color-primary-700': '158 123 42',
		'--color-primary-800': '126 98 32',
		'--color-primary-900': '95 74 22',

		// === Secondary: Silver/Platinum (#A8B4C4) ===
		'--color-secondary-50': '242 244 247',
		'--color-secondary-100': '229 233 240',
		'--color-secondary-200': '202 210 222',
		'--color-secondary-300': '168 180 196',   // #A8B4C4 base
		'--color-secondary-400': '140 156 176',
		'--color-secondary-500': '110 124 145',
		'--color-secondary-600': '88 100 118',
		'--color-secondary-700': '66 76 90',
		'--color-secondary-800': '44 50 60',
		'--color-secondary-900': '22 26 32',

		// === Tertiary: Champagne (#E8C96A) ===
		'--color-tertiary-50': '253 248 232',
		'--color-tertiary-100': '250 241 209',
		'--color-tertiary-200': '244 225 162',
		'--color-tertiary-300': '232 201 106',   // #E8C96A base
		'--color-tertiary-400': '220 183 73',
		'--color-tertiary-500': '196 158 50',
		'--color-tertiary-600': '157 126 38',
		'--color-tertiary-700': '118 95 28',
		'--color-tertiary-800': '79 63 18',
		'--color-tertiary-900': '40 32 9',

		// === Success: Muted green ===
		'--color-success-50': '221 244 228',
		'--color-success-100': '196 236 211',
		'--color-success-200': '145 218 167',
		'--color-success-300': '95 199 124',
		'--color-success-400': '60 185 92',
		'--color-success-500': '34 139 34',
		'--color-success-600': '27 111 27',
		'--color-success-700': '20 83 20',
		'--color-success-800': '14 55 14',
		'--color-success-900': '7 28 7',

		// === Warning: Gold-orange ===
		'--color-warning-50': '255 246 224',
		'--color-warning-100': '254 237 192',
		'--color-warning-200': '252 218 130',
		'--color-warning-300': '251 185 52',
		'--color-warning-400': '234 158 18',
		'--color-warning-500': '202 122 0',
		'--color-warning-600': '161 97 0',
		'--color-warning-700': '121 72 0',
		'--color-warning-800': '81 48 0',
		'--color-warning-900': '40 24 0',

		// === Error: Muted red ===
		'--color-error-50': '252 228 228',
		'--color-error-100': '249 202 202',
		'--color-error-200': '242 149 149',
		'--color-error-300': '235 96 96',
		'--color-error-400': '213 53 53',
		'--color-error-500': '185 20 20',
		'--color-error-600': '148 16 16',
		'--color-error-700': '111 12 12',
		'--color-error-800': '74 8 8',
		'--color-error-900': '37 4 4',

		// === Surface: Deep charcoal (#0C0E14) ===
		'--color-surface-50': '220 221 224',
		'--color-surface-100': '185 187 193',
		'--color-surface-200': '140 143 153',
		'--color-surface-300': '90 95 110',
		'--color-surface-400': '48 52 66',
		'--color-surface-500': '25 28 38',    // near-black surface
		'--color-surface-600': '18 20 28',
		'--color-surface-700': '14 15 22',
		'--color-surface-800': '10 11 17',
		'--color-surface-900': '6 7 10'       // ~#060707
	}
};
