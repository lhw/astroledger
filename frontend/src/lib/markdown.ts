/**
 * Render user-generated Markdown to safe HTML.
 *
 * Uses `marked` for parsing (works in both SSR and browser) with the HTML
 * renderer disabled so users cannot inject raw HTML tags. Sanitization then
 * runs in both SSR and browser paths to avoid server-rendered XSS.
 *
 * Usage in a .svelte component:
 *   {@html renderMarkdown(text)}
 */
import { Marked } from 'marked';
import DOMPurify from 'isomorphic-dompurify';

const ALLOWED_HTML_TAGS = [
	'p', 'strong', 'em', 'ul', 'ol', 'li', 'a', 'code', 'pre', 'blockquote', 'br',
	'h1', 'h2', 'h3', 'h4'
];

// Only absolute HTTPS links are allowed in rendered markdown.
const HTTPS_ONLY_URI = /^https:\/\/[\w.-]+(?:\:[0-9]{1,5})?(?:[/?#][^\s]*)?$/i;

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
	return DOMPurify.sanitize(html, {
		ALLOWED_TAGS: ALLOWED_HTML_TAGS,
		ALLOWED_URI_REGEXP: HTTPS_ONLY_URI,
		FORBID_ATTR: ['style', 'onerror', 'onclick', 'onload']
	});
}
