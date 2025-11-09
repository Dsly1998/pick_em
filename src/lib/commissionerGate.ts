const STORAGE_KEY = 'bdp:commishVerified';
const PASSCODE = 'godawgs';

function getWindow(): Window | undefined {
	if (typeof window === 'undefined') {
		return undefined;
	}
	return window;
}

export function isCommissionerVerified(): boolean {
	const win = getWindow();
	if (!win) return false;
	return win.localStorage.getItem(STORAGE_KEY) === 'true';
}

export function ensureCommissionerAccess(): boolean {
	const win = getWindow();
	if (!win) return false;
	if (isCommissionerVerified()) {
		return true;
	}
	const response = win.prompt('Commissioner check: type "GoDawgs" to open picks.');
	if (!response) {
		return false;
	}
	if (response.trim().toLowerCase() !== PASSCODE.toLowerCase()) {
		win.alert('Sorry, only Commissioner Brad can open picks.');
		return false;
	}
	win.localStorage.setItem(STORAGE_KEY, 'true');
	return true;
}

export function resetCommissionerAccess() {
	const win = getWindow();
	if (!win) return;
	win.localStorage.removeItem(STORAGE_KEY);
}
