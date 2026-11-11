import type { Row } from './rows';

export const pending = {
	'pending.title': ['Pending actions', 'Actions en attente', 'Acciones pendientes', 'Ausstehende Aktionen'],
	'pending.intro': [
		'An AI client asked Journal to do these. Nothing happens until you confirm.',
		'Un client IA a demandé ces actions à Journal. Rien ne se passe tant que vous ne confirmez pas.',
		'Un cliente de IA pidió estas acciones a Journal. No ocurre nada hasta que confirmes.',
		'Ein KI-Client hat Journal um diese Aktionen gebeten. Nichts passiert, bevor du bestätigst.'
	],
	'pending.none': ['Nothing is waiting', 'Rien en attente', 'Nada pendiente', 'Nichts ausstehend'],
	'pending.noneBody': [
		'Actions an AI client asks for appear here until you confirm or dismiss them.',
		'Les actions demandées par un client IA apparaissent ici jusqu’à ce que vous les confirmiez ou les ignoriez.',
		'Las acciones que pide un cliente de IA aparecen aquí hasta que las confirmes o descartes.',
		'Aktionen, um die ein KI-Client bittet, erscheinen hier, bis du sie bestätigst oder verwirfst.'
	],
	'pending.deleteQuery': [
		'Delete saved query',
		'Supprimer la requête',
		'Eliminar consulta guardada',
		'Gespeicherte Abfrage löschen'
	],
	'pending.deleteAlert': ['Delete alert rule', 'Supprimer l’alerte', 'Eliminar alerta', 'Warnregel löschen'],
	'pending.askedBy': ['Asked by {client}', 'Demandé par {client}', 'Pedido por {client}', 'Angefragt von {client}'],
	'pending.sharedQuery': [
		'Everyone in Journal loses this saved query.',
		'Tout le monde dans Journal perd cette requête enregistrée.',
		'Todos en Journal pierden esta consulta guardada.',
		'Alle in Journal verlieren diese gespeicherte Abfrage.'
	],
	'pending.alertRule': [
		'{threshold} entries in {minutes} min on {query}',
		'{threshold} entrées en {minutes} min sur {query}',
		'{threshold} entradas en {minutes} min en {query}',
		'{threshold} Einträge in {minutes} Min. für {query}'
	],
	'pending.alertOff': ['Disabled', 'Désactivée', 'Desactivada', 'Deaktiviert'],
	'pending.confirmDelete': ['Delete', 'Supprimer', 'Eliminar', 'Löschen'],
	'pending.dismiss': ['Dismiss', 'Ignorer', 'Descartar', 'Verwerfen'],
	'pending.changed': [
		'This changed after the request. Ask the AI client again.',
		'Cet élément a changé après la demande. Redemandez au client IA.',
		'Esto cambió después de la solicitud. Vuelve a pedírselo al cliente de IA.',
		'Das wurde nach der Anfrage geändert. Frag den KI-Client erneut.'
	],
	'pending.missing': [
		'This no longer exists.',
		'Cet élément n’existe plus.',
		'Esto ya no existe.',
		'Das existiert nicht mehr.'
	],
	'pending.deleted': ['Deleted', 'Supprimé', 'Eliminado', 'Gelöscht'],
	'pending.dismissed': [
		'Dismissed. Nothing was deleted.',
		'Ignoré. Rien n’a été supprimé.',
		'Descartado. No se eliminó nada.',
		'Verworfen. Nichts wurde gelöscht.'
	],
	'pending.failed': ['That did not work', 'Cela n’a pas marché', 'No ha funcionado', 'Das hat nicht geklappt'],
	'pending.loadFailed': [
		'Could not load pending actions',
		'Impossible de charger les actions en attente',
		'No se pudieron cargar las acciones pendientes',
		'Ausstehende Aktionen konnten nicht geladen werden'
	]
} satisfies Record<string, Row>;
