import { backend, setReadScope, type Workspace } from '$lib/backend';

const STORAGE_KEY = 'journal.spaces';

const state = $state<{ all: Workspace[]; selected: number[]; version: number }>({ all: [], selected: [], version: 0 });

function remembered(): number[] {
	try {
		const raw = JSON.parse(localStorage.getItem(STORAGE_KEY) ?? '[]');
		return Array.isArray(raw) ? raw.filter((id): id is number => Number.isInteger(id)) : [];
	} catch {
		return [];
	}
}

function remember(ids: number[]) {
	try {
		localStorage.setItem(STORAGE_KEY, JSON.stringify(ids));
	} catch {
		return;
	}
}

/** The caller's workspaces and the ones the explorer reads now. No selection means every one. */
export const spaces = {
	get all() {
		return state.all;
	},
	get selected() {
		return state.selected;
	},
	get version() {
		return state.version;
	},
	/** The ids a read sends, or undefined for every workspace. */
	get scope(): number[] | undefined {
		return state.selected.length ? state.selected : undefined;
	},
	/** Whether the current view spans more than one workspace, so rows name theirs. */
	get multi() {
		return (state.selected.length || state.all.length) > 1;
	},
	get owned() {
		return state.all.filter((w) => w.role === 'owner');
	},
	/** The one workspace a write goes to when the view holds exactly one. */
	get target(): number | undefined {
		if (state.selected.length === 1) return state.selected[0];
		return state.all.length === 1 ? state.all[0].id : undefined;
	},
	name(id: number) {
		return state.all.find((w) => w.id === id)?.name ?? '';
	},
	async load() {
		state.all = (await backend.listWorkspaces()).workspaces;
		const known = new Set(state.all.map((w) => w.id));
		state.selected = remembered().filter((id) => known.has(id));
		setReadScope(state.selected);
	},
	select(ids: number[]) {
		const unique = [...new Set(ids)].filter((id) => state.all.some((w) => w.id === id));
		state.selected = unique.length === state.all.length ? [] : unique;
		remember(state.selected);
		setReadScope(state.selected);
		state.version += 1;
	}
};
