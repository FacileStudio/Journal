import type { Row } from './rows';

export const chrome = {
	'app.tagline': [
		'Centralized logging for every app in the Facile Suite.',
		'La journalisation centralisée de toutes les applications de la suite Facile.',
		'Registro centralizado para todas las aplicaciones de la suite Facile.',
		'Zentrales Logging für alle Apps der Facile-Suite.'
	],
	'app.titleSuffix': ['Journal', 'Journal', 'Journal', 'Journal'],
	'login.title': ['Log in', 'Connexion', 'Iniciar sesión', 'Anmelden'],
	'nav.overview': ['Overview', 'Aperçu', 'Resumen', 'Übersicht'],
	'nav.logs': ['Logs', 'Journaux', 'Registros', 'Logs'],
	'nav.apps': ['Apps', 'Applications', 'Aplicaciones', 'Apps'],
	'nav.queries': ['Queries', 'Requêtes', 'Consultas', 'Abfragen'],
	'nav.alerts': ['Alerts', 'Alertes', 'Alertas', 'Warnungen'],
	'nav.account': ['Account', 'Compte', 'Cuenta', 'Konto'],
	'common.cancel': ['Cancel', 'Annuler', 'Cancelar', 'Abbrechen'],
	'common.delete': ['Delete', 'Supprimer', 'Eliminar', 'Löschen'],
	'common.save': ['Save', 'Enregistrer', 'Guardar', 'Speichern'],
	'common.saving': ['Saving…', 'Enregistrement…', 'Guardando…', 'Speichern…'],
	'common.creating': ['Creating…', 'Création…', 'Creando…', 'Wird erstellt…'],
	'common.loading': ['Loading…', 'Chargement…', 'Cargando…', 'Wird geladen…'],
	'common.copy': ['Copy', 'Copier', 'Copiar', 'Kopieren'],
	'common.copied': ['Copied', 'Copié', 'Copiado', 'Kopiert'],
	'common.done': ['Done', 'Terminé', 'Listo', 'Fertig'],
	'common.never': ['never', 'jamais', 'nunca', 'nie'],
	'common.actions': ['Actions', 'Actions', 'Acciones', 'Aktionen'],
	'common.app': ['App', 'Application', 'Aplicación', 'App'],
	'common.entries': ['Entries', 'Entrées', 'Entradas', 'Einträge'],
	'common.lastSeen': ['Last seen', 'Dernière activité', 'Última actividad', 'Zuletzt gesehen'],
	'common.created': ['Created', 'Créé', 'Creado', 'Erstellt'],
	'common.name': ['Name', 'Nom', 'Nombre', 'Name'],
	'common.level': ['Level', 'Niveau', 'Nivel', 'Stufe'],
	'common.status': ['Status', 'État', 'Estado', 'Status'],
	'common.unknownError': ['Something went wrong', 'Une erreur est survenue', 'Algo salió mal', 'Etwas ist schiefgelaufen'],
	'clipboard.copied': [
		'Entry context copied to clipboard.',
		'Contexte de l’entrée copié dans le presse-papiers.',
		'Contexto de la entrada copiado al portapapeles.',
		'Eintragskontext in die Zwischenablage kopiert.'
	],
	'clipboard.failed': [
		'Could not copy to clipboard.',
		'Impossible de copier dans le presse-papiers.',
		'No se pudo copiar al portapapeles.',
		'Kopieren in die Zwischenablage fehlgeschlagen.'
	],
	'clipboard.unavailable': [
		'Clipboard is not available in this browser.',
		'Le presse-papiers n’est pas disponible dans ce navigateur.',
		'El portapapeles no está disponible en este navegador.',
		'Die Zwischenablage ist in diesem Browser nicht verfügbar.'
	],
	'overview.title': ['Overview', 'Aperçu', 'Resumen', 'Übersicht'],
	'overview.description': [
		'What every app in the suite has been saying for the last 24 hours.',
		'Ce que chaque application de la suite a journalisé ces dernières 24 heures.',
		'Lo que cada aplicación de la suite ha registrado en las últimas 24 horas.',
		'Was jede App der Suite in den letzten 24 Stunden protokolliert hat.'
	],
	'overview.openLogs': ['Open logs', 'Ouvrir les journaux', 'Abrir los registros', 'Logs öffnen'],
	'overview.stat.entries': ['Entries', 'Entrées', 'Entradas', 'Einträge'],
	'overview.stat.errors': ['Errors', 'Erreurs', 'Errores', 'Fehler'],
	'overview.stat.warnings': ['Warnings', 'Avertissements', 'Advertencias', 'Warnungen'],
	'overview.stat.apps': ['Apps reporting', 'Applications actives', 'Aplicaciones activas', 'Berichtende Apps'],
	'overview.volume': ['Volume', 'Volume', 'Volumen', 'Volumen'],
	'overview.volumeHint': [
		'Click a bucket to open it in the log explorer.',
		'Cliquez sur une barre pour l’ouvrir dans l’explorateur.',
		'Haga clic en una barra para abrirla en el explorador.',
		'Klicken Sie auf einen Balken, um ihn im Explorer zu öffnen.'
	],
	'overview.byLevel': ['By level', 'Par niveau', 'Por nivel', 'Nach Stufe'],
	'overview.centerLabel': ['entries', 'entrées', 'entradas', 'Einträge'],
	'overview.nothing': ['Nothing ingested yet', 'Rien n’a encore été reçu', 'Aún no se ha recibido nada', 'Noch nichts empfangen'],
	'overview.busiest': ['Busiest apps', 'Applications les plus actives', 'Aplicaciones más activas', 'Aktivste Apps'],
	'overview.noApp': [
		'No app has shipped a log yet',
		'Aucune application n’a encore envoyé de journal',
		'Ninguna aplicación ha enviado registros todavía',
		'Noch keine App hat Logs gesendet'
	],
	'overview.recentErrors': ['Recent errors', 'Erreurs récentes', 'Errores recientes', 'Letzte Fehler'],
	'overview.lastEight': [
		'The last eight, across every app.',
		'Les huit dernières, toutes applications confondues.',
		'Las ocho últimas, de todas las aplicaciones.',
		'Die letzten acht, über alle Apps hinweg.'
	],
	'overview.seeAllErrors': ['See all errors', 'Voir toutes les erreurs', 'Ver todos los errores', 'Alle Fehler anzeigen'],
	'overview.noErrors': [
		'No errors in the last 24 hours',
		'Aucune erreur ces dernières 24 heures',
		'Ningún error en las últimas 24 horas',
		'Keine Fehler in den letzten 24 Stunden'
	],
	'overview.noErrorsHint': [
		'Quiet is good. The explorer has the full stream if you want to look closer.',
		'Le calme est bon signe. L’explorateur contient le flux complet si vous voulez regarder de plus près.',
		'La calma es buena señal. El explorador tiene el flujo completo si quiere mirar más de cerca.',
		'Ruhe ist gut. Der Explorer zeigt den vollständigen Verlauf, falls Sie genauer hinsehen möchten.'
	],
	'apps.title': ['Apps', 'Applications', 'Aplicaciones', 'Apps'],
	'apps.description': [
		'Every source that has ever shipped an entry to this instance.',
		'Chaque source qui a déjà envoyé une entrée à cette instance.',
		'Cada fuente que ha enviado alguna vez una entrada a esta instancia.',
		'Jede Quelle, die jemals einen Eintrag an diese Instanz gesendet hat.'
	],
	'apps.loadFailed': ['Could not load apps', 'Impossible de charger les applications', 'No se pudieron cargar las aplicaciones', 'Apps konnten nicht geladen werden'],
	'apps.emptyHint': [
		'Create an ingest key, point the app’s JOURNAL_URL at this instance, and its first entry will name it here.',
		'Créez une clé d’ingestion, pointez le JOURNAL_URL de l’application vers cette instance, et sa première entrée la fera apparaître ici.',
		'Cree una clave de ingesta, apunte el JOURNAL_URL de la aplicación a esta instancia y su primera entrada la nombrará aquí.',
		'Erstellen Sie einen Ingest-Schlüssel und richten Sie die JOURNAL_URL der App auf diese Instanz; ihr erster Eintrag erscheint dann hier.'
	],
	'apps.createKey': ['Create an ingest key', 'Créer une clé d’ingestion', 'Crear una clave de ingesta', 'Ingest-Schlüssel erstellen'],
	'apps.logsLink': ['Logs', 'Journaux', 'Registros', 'Logs']
} as const satisfies Record<string, Row>;
