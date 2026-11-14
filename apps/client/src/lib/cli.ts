import type { CliCommand, Translator } from '@facile/muse';
import type { Key } from '$lib/i18n';

const REPO = 'https://github.com/FacileStudio/journal-cli';

export const CLI_RELEASES_URL = `${REPO}/releases`;
export const CLI_CHECKSUMS_URL = `${REPO}/releases/latest/download/checksums.txt`;

/** The journal CLI's install commands, as its README and the FacileStudio Homebrew tap give them. */
export function cliCommands(t: Translator<Record<Key, string>>): CliCommand[] {
	return [
		{ label: t('downloads.install.script'), command: 'curl -fsSL https://raw.githubusercontent.com/FacileStudio/journal-cli/main/install.sh | bash' },
		{ label: t('downloads.install.facile'), command: 'facile install journal' },
		{ label: t('downloads.install.brew'), command: 'brew install --cask FacileStudio/tap/journal' }
	];
}
