import type { Row } from './rows';

export const spaces = {
	'spaces.label': ['Spaces', 'Espaces', 'Espacios', 'Bereiche'],
	'spaces.all': ['All spaces', 'Tous les espaces', 'Todos los espacios', 'Alle Bereiche'],
	'spaces.some': ['{count} spaces', '{count} espaces', '{count} espacios', '{count} Bereiche'],
	'spaces.personal': ['Personal', 'Personnel', 'Personal', 'Persönlich'],
	'spaces.hint': [
		'Tick several spaces to read their logs side by side.',
		'Cochez plusieurs espaces pour lire leurs journaux côte à côte.',
		'Marque varios espacios para leer sus registros juntos.',
		'Wählen Sie mehrere Bereiche, um ihre Logs nebeneinander zu lesen.'
	],
	'spaces.target': ['Space', 'Espace', 'Espacio', 'Bereich'],
	'spaces.emptyTitle': ['No logs in this space yet', 'Aucun journal dans cet espace', 'Aún no hay registros en este espacio', 'Noch keine Logs in diesem Bereich'],
	'spaces.emptyHint': [
		'Create an API key in Settings, then send your first entry. Your logs stay in your space: nobody else can read them.',
		'Créez une clé d’API dans les réglages, puis envoyez votre première entrée. Vos journaux restent dans votre espace : personne d’autre ne peut les lire.',
		'Cree una clave de API en Ajustes y envíe su primera entrada. Sus registros se quedan en su espacio: nadie más puede leerlos.',
		'Erstellen Sie in den Einstellungen einen API-Schlüssel und senden Sie Ihren ersten Eintrag. Ihre Logs bleiben in Ihrem Bereich: niemand sonst kann sie lesen.'
	],
	'spaces.createKey': ['Create an API key', 'Créer une clé d’API', 'Crear una clave de API', 'API-Schlüssel erstellen']
} as const satisfies Record<string, Row>;
