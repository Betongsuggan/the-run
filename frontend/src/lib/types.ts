export type ID = string;

export type Gender = 'M' | 'F' | 'X';

export type Discipline = 'run' | 'walk' | 'kids';

export type Runner = {
	id: ID;
	name: string;
	gender: Gender;
	birthYear?: number;
};

export type RaceEvent = {
	id: ID;
	name: string;
	year: number;
	date: string;
	location?: string;
};

export type Race = {
	id: ID;
	eventId: ID;
	name: string;
	distanceMeters: number;
	discipline: Discipline;
};

export type Category = {
	gender?: Gender;
	ageGroup?: string;
};

export type Split = {
	km: number;
	timeSeconds: number;
};

export type RegistrationStatus = 'pending' | 'finished' | 'dnf' | 'dns';

export type Result = {
	id: ID;
	raceId: ID;
	runnerId: ID;
	// "finished" rows carry finishSeconds + placement; "dnf"/"dns" rows
	// have neither (server sends finishSeconds as undefined). Frontend
	// branches on status to render a badge instead of a time.
	status: RegistrationStatus;
	bib: string;
	finishSeconds?: number;
	category: Category;
	placementOverall?: number;
	placementCategory?: number;
	splits?: Split[];
	conditions?: string;
	notes?: string;
};

export type Registration = {
	id: ID;
	raceId: ID;
	runnerId: ID;
	bib?: string;
	category?: Category;
	status: RegistrationStatus;
	finishSeconds?: number;
	splits?: Split[];
	conditions?: string;
	notes?: string;
};

export type ResultExpanded = Result & {
	race: Race;
	event: RaceEvent;
	runner: Runner;
};

export type PolicyStatus = 'draft' | 'published' | 'archived';

// PolicyKind identifies which user-facing document a Policy row represents.
// Currently only the privacy policy is defined; the type stays a plain string
// to keep extension friction-free as we add ToS / Code of Conduct / etc.
export type PolicyKind = 'privacy';

// Versioned policy document. The body is admin-authored markdown; the `slug`
// is the human-facing version label users see (e.g. "2026-08-01"). `revision`
// bumps on in-place edits of a published policy and is part of the FK
// referenced by each Consent record.
export type Policy = {
	id: ID;
	kind: PolicyKind;
	slug: string;
	status: PolicyStatus;
	revision: number;
	effectiveFrom: string;
	bodySv: string;
	bodyEn: string;
	publishedAt?: string;
	updatedAt?: string;
};

// One snapshot in the edit history of a policy. Returned from
// /admin/policies/{id}/revisions and /policies/{id}/revisions/{n}; the
// body is the exact text that was live at that revision.
export type PolicyRevision = {
	policyId: ID;
	revision: number;
	bodySv: string;
	bodyEn: string;
	editedAt: string;
	note?: string;
	published: boolean;
};
