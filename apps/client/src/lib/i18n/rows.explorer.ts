import type { Row } from './rows';

export const explorer = {
	'logs.title': ['Logs', 'Journaux', 'Registros', 'Logs'],
	'logs.description': [
		'Every entry shipped to this instance, newest first.',
		'Chaque entrée envoyée à cette instance, la plus récente en premier.',
		'Cada entrada enviada a esta instancia, la más reciente primero.',
		'Jeder an diese Instanz gesendete Eintrag, neueste zuerst.'
	],
	'logs.resume': ['Resume', 'Reprendre', 'Reanudar', 'Fortsetzen'],
	'logs.pause': ['Pause', 'Pause', 'Pausa', 'Pause'],
	'logs.liveTail': ['Live tail', 'Suivi en direct', 'Seguimiento en vivo', 'Live-Verlauf'],
	'logs.saveFilters': ['Save filters', 'Enregistrer les filtres', 'Guardar filtros', 'Filter speichern'],
	'logs.search': ['Search messages…', 'Rechercher dans les messages…', 'Buscar en los mensajes…', 'Nachrichten durchsuchen…'],
	'logs.searchLabel': ['Search messages', 'Rechercher dans les messages', 'Buscar en los mensajes', 'Nachrichten durchsuchen'],
	'logs.filterApp': ['Filter by app', 'Filtrer par application', 'Filtrar por aplicación', 'Nach App filtern'],
	'logs.allApps': ['All apps', 'Toutes les applications', 'Todas las aplicaciones', 'Alle Apps'],
	'logs.filterSource': ['Filter by source', 'Filtrer par source', 'Filtrar por origen', 'Nach Quelle filtern'],
	'logs.source': ['Source', 'Source', 'Origen', 'Quelle'],
	'logs.allSources': ['All sources', 'Toutes les sources', 'Todos los orígenes', 'Alle Quellen'],
	'logs.sourceBrowser': ['Client (browser)', 'Client (navigateur)', 'Cliente (navegador)', 'Client (Browser)'],
	'logs.sourceCollector': ['Collector (Docker)', 'Collecteur (Docker)', 'Colector (Docker)', 'Collector (Docker)'],
	'logs.sourceServer': ['Server (no source)', 'Serveur (sans source)', 'Servidor (sin origen)', 'Server (ohne Quelle)'],
	'logs.savedQuery': ['Saved query', 'Requête enregistrée', 'Consulta guardada', 'Gespeicherte Abfrage'],
	'logs.applySaved': ['Apply a saved query', 'Appliquer une requête enregistrée', 'Aplicar una consulta guardada', 'Gespeicherte Abfrage anwenden'],
	'logs.timeRange': ['Time range', 'Période', 'Intervalo de tiempo', 'Zeitraum'],
	'logs.rangeAll': ['All', 'Tout', 'Todo', 'Alle'],
	'logs.rangeCustom': ['Custom', 'Personnalisé', 'Personalizado', 'Benutzerdefiniert'],
	'logs.from': ['From', 'Du', 'Desde', 'Von'],
	'logs.to': ['To', 'Au', 'Hasta', 'Bis'],
	'logs.clearFilters': ['Clear filters ({count})', 'Effacer les filtres ({count})', 'Quitar filtros ({count})', 'Filter zurücksetzen ({count})'],
	'logs.loadFailed': ['Could not load logs', 'Impossible de charger les journaux', 'No se pudieron cargar los registros', 'Logs konnten nicht geladen werden'],
	'logs.loadMoreFailed': ['Failed to load more', 'Impossible de charger la suite', 'No se pudo cargar más', 'Weitere Einträge konnten nicht geladen werden'],
	'logs.emptyHint': [
		'Widen the time range, drop a level filter, or check that the app is shipping to this instance.',
		'Élargissez la période, retirez un filtre de niveau ou vérifiez que l’application envoie bien ses journaux à cette instance.',
		'Amplíe el intervalo, quite un filtro de nivel o compruebe que la aplicación envía registros a esta instancia.',
		'Erweitern Sie den Zeitraum, entfernen Sie einen Stufenfilter oder prüfen Sie, ob die App an diese Instanz sendet.'
	],
	'logs.loadMore': ['Load more', 'Charger la suite', 'Cargar más', 'Mehr laden'],
	'logs.saveTitle': ['Save these filters', 'Enregistrer ces filtres', 'Guardar estos filtros', 'Diese Filter speichern'],
	'logs.saveNamePlaceholder': ['Nuage errors', 'Erreurs Nuage', 'Errores de Nuage', 'Nuage-Fehler'],
	'logs.saved': ['Saved “{name}”.', '« {name} » enregistrée.', '«{name}» guardada.', '„{name}“ gespeichert.'],
	'logs.saveFailed': ['Could not save the query', 'Impossible d’enregistrer la requête', 'No se pudo guardar la consulta', 'Abfrage konnte nicht gespeichert werden'],
	'table.time': ['Time', 'Heure', 'Hora', 'Zeit'],
	'table.message': ['Message', 'Message', 'Mensaje', 'Nachricht'],
	'table.empty': [
		'No log entries match these filters',
		'Aucune entrée ne correspond à ces filtres',
		'Ninguna entrada coincide con estos filtros',
		'Keine Einträge entsprechen diesen Filtern'
	],
	'table.filterBy': ['Filter by {value}', 'Filtrer par {value}', 'Filtrar por {value}', 'Nach {value} filtern'],
	'table.filterBySource': ['Filter by source: {value}', 'Filtrer par source : {value}', 'Filtrar por origen: {value}', 'Nach Quelle filtern: {value}'],
	'table.sourceBrowser': ['browser', 'navigateur', 'navegador', 'Browser'],
	'table.sourceCollector': ['collector', 'collecteur', 'colector', 'Collector'],
	'table.sourceServer': ['server', 'serveur', 'servidor', 'Server'],
	'table.frameMapped': [
		'{resolved} of {total} frames mapped from release',
		'{resolved} cadres sur {total} résolus depuis la version',
		'{resolved} de {total} marcos resueltos desde la versión',
		'{resolved} von {total} Frames aus Release'
	],
	'table.noSourceMap': [
		'No source map uploaded for release',
		'Aucune source map envoyée pour la version',
		'No se subió ningún source map para la versión',
		'Keine Source Map für Release hochgeladen'
	],
	'table.noMeta': ['No metadata.', 'Aucune métadonnée.', 'Sin metadatos.', 'Keine Metadaten.'],
	'table.context': ['Context', 'Contexte', 'Contexto', 'Kontext'],
	'table.received': ['received', 'reçu', 'recibido', 'empfangen'],
	'table.gap': [
		'possible gap — some entries between these rows were never fetched',
		'trou possible — certaines entrées entre ces lignes n’ont jamais été chargées',
		'posible hueco — algunas entradas entre estas filas nunca se cargaron',
		'möglicher Abstand — einige Einträge zwischen diesen Zeilen wurden nie geladen'
	],
	'drawer.title': ['Log entry', 'Entrée de journal', 'Entrada de registro', 'Log-Eintrag'],
	'drawer.copyContext': ['Copy entry context', 'Copier le contexte de l’entrée', 'Copiar el contexto de la entrada', 'Eintragskontext kopieren'],
	'drawer.loadFailed': ['Could not load this entry', 'Impossible de charger cette entrée', 'No se pudo cargar esta entrada', 'Dieser Eintrag konnte nicht geladen werden'],
	'drawer.created': ['Created', 'Créé', 'Creado', 'Erstellt'],
	'drawer.receivedLabel': ['Received', 'Reçu', 'Recibido', 'Empfangen'],
	'drawer.message': ['Message', 'Message', 'Mensaje', 'Nachricht'],
	'drawer.meta': ['Meta', 'Métadonnées', 'Metadatos', 'Metadaten'],
	'histogram.noVolume': ['No volume in this range', 'Aucun volume sur cette période', 'Sin volumen en este intervalo', 'Kein Volumen in diesem Zeitraum'],
	'histogram.caption': [
		'Log volume by level over the selected range',
		'Volume de journaux par niveau sur la période choisie',
		'Volumen de registros por nivel en el intervalo seleccionado',
		'Log-Volumen nach Stufe im gewählten Zeitraum'
	],
	'histogram.bucket': ['Bucket', 'Intervalle', 'Intervalo', 'Zeitfenster'],
	'histogram.none': ['no entries', 'aucune entrée', 'sin entradas', 'keine Einträge'],
	'queries.title': ['Saved queries', 'Requêtes enregistrées', 'Consultas guardadas', 'Gespeicherte Abfragen'],
	'queries.description': [
		'Named filter sets. Alert rules are built on top of them.',
		'Des jeux de filtres nommés. Les règles d’alerte s’appuient dessus.',
		'Conjuntos de filtros con nombre. Las reglas de alerta se basan en ellos.',
		'Benannte Filtersätze. Warnregeln bauen darauf auf.'
	],
	'queries.saveNew': ['Save a new one', 'En enregistrer une', 'Guardar una nueva', 'Neue speichern'],
	'queries.loadFailed': ['Could not load saved queries', 'Impossible de charger les requêtes enregistrées', 'No se pudieron cargar las consultas guardadas', 'Gespeicherte Abfragen konnten nicht geladen werden'],
	'queries.empty': ['No saved queries yet', 'Aucune requête enregistrée', 'Aún no hay consultas guardadas', 'Noch keine gespeicherten Abfragen'],
	'queries.emptyHint': [
		'Set up a filter in the explorer, then save it — an alert rule can only watch a query that has a name.',
		'Réglez un filtre dans l’explorateur puis enregistrez-le : une règle d’alerte ne peut surveiller qu’une requête nommée.',
		'Configure un filtro en el explorador y guárdelo: una regla de alerta solo puede vigilar una consulta con nombre.',
		'Richten Sie im Explorer einen Filter ein und speichern Sie ihn: Eine Warnregel kann nur eine benannte Abfrage überwachen.'
	],
	'queries.openExplorer': ['Open the explorer', 'Ouvrir l’explorateur', 'Abrir el explorador', 'Explorer öffnen'],
	'queries.filters': ['Filters', 'Filtres', 'Filtros', 'Filter'],
	'queries.noFilters': ['no filters', 'aucun filtre', 'sin filtros', 'keine Filter'],
	'queries.deleted': ['Deleted “{name}”.', '« {name} » supprimée.', '«{name}» eliminada.', '„{name}“ gelöscht.'],
	'queries.deleteFailed': ['Could not delete the query', 'Impossible de supprimer la requête', 'No se pudo eliminar la consulta', 'Abfrage konnte nicht gelöscht werden'],
	'queries.deleteTitle': ['Delete this saved query?', 'Supprimer cette requête enregistrée ?', '¿Eliminar esta consulta guardada?', 'Diese gespeicherte Abfrage löschen?'],
	'queries.deleteBody': [
		'Any alert rule built on it must be deleted first, and the filter set itself cannot be recovered.',
		'Toute règle d’alerte qui s’appuie dessus doit d’abord être supprimée, et le jeu de filtres est irrécupérable.',
		'Toda regla de alerta basada en ella debe eliminarse antes, y el conjunto de filtros no se puede recuperar.',
		'Jede darauf basierende Warnregel muss zuerst gelöscht werden, und der Filtersatz selbst lässt sich nicht wiederherstellen.'
	]
} as const satisfies Record<string, Row>;
