/**
 * Render user-generated Markdown to safe HTML.
 *
 * Uses `marked` for parsing (works in both SSR and browser) with the HTML
 * renderer disabled so users cannot inject raw HTML tags.  DOMPurify then
 * provides a second layer of defence in the browser.
 *
 * Usage in a .svelte component:
 *   {@html renderMarkdown(text)}
 */
import { browser } from '$app/environment';
import { Marked } from 'marked';
import DOMPurify from 'dompurify';

const ALLOWED_HTML_TAGS = [
	'p', 'strong', 'em', 'ul', 'ol', 'li', 'a', 'code', 'pre', 'blockquote', 'br',
	'h1', 'h2', 'h3', 'h4'
];

// Configure marked to strip raw HTML input — only safe markdown constructs pass through.
const md = new Marked();
md.use({
	renderer: {
		html() {
			return ''; // Drop raw HTML blocks/inline that users might sneak in.
		}
	}
});

export function renderMarkdown(raw: string): string {
	if (!raw) return '';
	const html = md.parse(raw, { async: false }) as string;
	// In the browser, also sanitize with DOMPurify for defence-in-depth.
	if (browser) {
		return DOMPurify.sanitize(html, { ALLOWED_TAGS: ALLOWED_HTML_TAGS });
	}
	return html;
}
