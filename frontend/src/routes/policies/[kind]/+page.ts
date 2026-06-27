import type { EntryGenerator } from './$types';

// The static adapter needs to know which `kind` values to prerender. The
// list mirrors the PolicyKind enum on the backend. Adding a new kind there
// is a coordinated change here too.
export const entries: EntryGenerator = () => {
	return [{ kind: 'privacy' }];
};

export const prerender = true;
