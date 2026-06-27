// Renders privacy-policy markdown safely. The body of every policy is
// admin-authored markdown stored on the server; we parse it with `marked`
// (sync mode — no async tokens, no remote fetch) and then run the output
// through DOMPurify with a tight allowlist so a malicious admin can't
// inject scripts. See B0.1 in the GDPR plan.
//
// Used by:
//   - /privacy (frontend/src/routes/privacy/+page.svelte) — public render
//   - admin policy preview (frontend/src/routes/admin/policies/+page.svelte)
//   - my-data deep-link to a historical revision

import DOMPurify from 'dompurify';
import { marked } from 'marked';

// Configure marked once. GFM tables + breaks would be nice; we leave them
// off to keep the surface area minimal.
marked.setOptions({
	gfm: true,
	breaks: false
});

// Allowed tags + attributes. Headings, paragraphs, lists, inline emphasis,
// links, blockquotes and GFM tables — enough for a privacy notice (the
// retention windows + lawful-basis matrix read naturally as tables) without
// opening the door to arbitrary HTML. Anything richer belongs in a real CMS.
const SANITIZE_OPTS = {
	ALLOWED_TAGS: [
		'h1',
		'h2',
		'h3',
		'h4',
		'p',
		'br',
		'strong',
		'em',
		'code',
		'pre',
		'ul',
		'ol',
		'li',
		'a',
		'blockquote',
		'hr',
		'table',
		'thead',
		'tbody',
		'tr',
		'th',
		'td'
	],
	ALLOWED_ATTR: ['href', 'title', 'align'],
	ALLOW_DATA_ATTR: false
};

/** Parse markdown and return sanitized HTML ready for {@html …}. */
export function renderPolicyMarkdown(markdown: string): string {
	const raw = marked.parse(markdown, { async: false }) as string;
	const clean = DOMPurify.sanitize(raw, SANITIZE_OPTS);
	// Force-add rel attrs to external links so a malicious link can't open in
	// the same tab and grab `window.opener`. DOMPurify's hook system is
	// heavier than this needs to be.
	return clean.replace(/<a\s+([^>]*?)>/gi, (match, attrs) => {
		if (/rel\s*=/.test(attrs)) return match;
		return `<a ${attrs} rel="noopener nofollow">`;
	});
}
