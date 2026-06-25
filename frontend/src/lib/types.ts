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
