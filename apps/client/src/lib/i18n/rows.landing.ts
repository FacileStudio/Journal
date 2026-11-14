import type { Row } from './rows';

export const landing = {
	'landing.title': [
		'Journal · Logs for every Facile app',
		'Journal · Les journaux de chaque app Facile',
		'Journal · Los registros de cada app Facile',
		'Journal · Logs für jede Facile-App'
	],
	'landing.nav.features': ['Features', 'Fonctions', 'Funciones', 'Funktionen'],
	'landing.nav.download': ['CLI', 'CLI', 'CLI', 'CLI'],
	'landing.hero.title': [
		'Every log, one place',
		'Tous vos journaux, au même endroit',
		'Todos los registros, en un solo lugar',
		'Alle Logs an einem Ort'
	],
	'landing.hero.body': [
		'Collect, search and follow the logs of every app, live.',
		'Collectez, cherchez et suivez en direct les journaux de chaque app.',
		'Recoge, busca y sigue en directo los registros de cada app.',
		'Sammeln, durchsuchen und live verfolgen: die Logs jeder App.'
	],
	'landing.hero.cli': ['Get the CLI', 'Installer la CLI', 'Instalar la CLI', 'CLI installieren'],
	'landing.demo.label': [
		'A terminal follows new log lines while the dashboard filters down to errors.',
		'Un terminal suit les nouvelles lignes pendant que le tableau de bord ne garde que les erreurs.',
		'Un terminal sigue las líneas nuevas mientras el panel filtra solo los errores.',
		'Ein Terminal folgt neuen Zeilen, während das Dashboard nur die Fehler zeigt.'
	],
	'landing.demo.search': ['Search logs', 'Chercher dans les journaux', 'Buscar registros', 'Logs durchsuchen'],
	'landing.demo.all': ['all', 'tout', 'todo', 'alle'],
	'landing.cards.title': ['From a crash to its cause', 'Du plantage à sa cause', 'Del fallo a su causa', 'Vom Absturz zur Ursache'],
	'landing.cards.search.title': ['Search every app', 'Chercher dans chaque app', 'Busca en cada app', 'Jede App durchsuchen'],
	'landing.cards.search.body': [
		'One query reads the logs of the whole suite.',
		'Une seule recherche lit les journaux de toute la suite.',
		'Una sola búsqueda lee los registros de toda la suite.',
		'Eine Suche liest die Logs der ganzen Suite.'
	],
	'landing.cards.alert.title': ['Alerts on spikes', 'Alertes sur les pics', 'Alertas en los picos', 'Warnungen bei Spitzen'],
	'landing.cards.alert.body': [
		'A rule fires when errors pass its threshold.',
		'Une règle se déclenche quand les erreurs dépassent son seuil.',
		'Una regla salta cuando los errores pasan su umbral.',
		'Eine Regel löst aus, wenn Fehler ihre Schwelle überschreiten.'
	],
	'landing.cards.alert.note': ['Alert fired', 'Alerte déclenchée', 'Alerta disparada', 'Warnung ausgelöst'],
	'landing.cards.stack.title': ['Readable stack traces', 'Traces lisibles', 'Trazas legibles', 'Lesbare Stacktraces'],
	'landing.cards.stack.body': [
		'Source maps turn minified frames back into your code.',
		'Les source maps ramènent les cadres minifiés à votre code.',
		'Los source maps devuelven los marcos minificados a tu código.',
		'Source Maps machen aus minifizierten Frames wieder deinen Code.'
	],
	'landing.features.collect.title': ['Collect without code', 'Collecte sans code', 'Recogida sin código', 'Sammeln ohne Code'],
	'landing.features.collect.body': [
		'A Docker collector ships container logs, and SDKs cover Go and the browser.',
		'Un collecteur Docker envoie les journaux des conteneurs, et les SDK couvrent Go et le navigateur.',
		'Un colector Docker envía los registros de los contenedores, y los SDK cubren Go y el navegador.',
		'Ein Docker-Collector liefert Container-Logs, SDKs decken Go und den Browser ab.'
	],
	'landing.features.search.title': ['Full-text search', 'Recherche plein texte', 'Búsqueda de texto completo', 'Volltextsuche'],
	'landing.features.search.body': [
		'Filter by app, level, time or request id, then search the message text.',
		'Filtrez par app, niveau, période ou identifiant de requête, puis cherchez dans le message.',
		'Filtra por app, nivel, hora o id de petición, y busca en el texto del mensaje.',
		'Nach App, Stufe, Zeit oder Request-ID filtern, dann im Text suchen.'
	],
	'landing.features.tail.title': ['Live tail', 'Suivi en direct', 'Seguimiento en directo', 'Live-Ansicht'],
	'landing.features.tail.body': [
		'New entries land as they happen, in the browser or the terminal.',
		'Les nouvelles entrées arrivent en direct, dans le navigateur ou le terminal.',
		'Las entradas nuevas llegan al momento, en el navegador o en el terminal.',
		'Neue Einträge erscheinen sofort, im Browser oder im Terminal.'
	],
	'landing.features.alerts.title': ['Alerts on errors', 'Alertes sur les erreurs', 'Alertas de errores', 'Warnungen bei Fehlern'],
	'landing.features.alerts.body': [
		'A rule sends an alert through Antenne before your users notice.',
		'Une règle envoie une alerte par Antenne avant que vos utilisateurs ne s’en aperçoivent.',
		'Una regla envía una alerta por Antenne antes de que tus usuarios lo noten.',
		'Eine Regel schickt über Antenne eine Warnung, bevor Ihre Nutzer es merken.'
	],
	'landing.download.heading': ['Logs in your terminal', 'Les journaux dans votre terminal', 'Registros en tu terminal', 'Logs im Terminal'],
	'landing.download.intro': [
		'The journal CLI queries and follows logs, even over SSH.',
		'La CLI journal interroge et suit les journaux, même en SSH.',
		'La CLI journal consulta y sigue los registros, incluso por SSH.',
		'Die journal-CLI fragt Logs ab und folgt ihnen, auch über SSH.'
	],
	'landing.cta.title': ['Start reading your logs', 'Lisez vos journaux', 'Empieza a leer tus registros', 'Logs jetzt lesen'],
	'landing.cta.body': [
		'Sign in with your Facile account.',
		'Connectez-vous avec votre compte Facile.',
		'Inicia sesión con tu cuenta Facile.',
		'Mit Ihrem Facile-Konto anmelden.'
	],
	'landing.footer.product': ['Product', 'Produit', 'Producto', 'Produkt'],
	'landing.footer.downloads': ['All downloads', 'Tous les téléchargements', 'Todas las descargas', 'Alle Downloads'],
	'landing.footer.docs': ['Documentation', 'Documentation', 'Documentación', 'Dokumentation'],
	'downloads.title': ['Downloads', 'Téléchargements', 'Descargas', 'Downloads'],
	'downloads.intro': [
		'Install the journal CLI on macOS or Linux.',
		'Installez la CLI journal sur macOS ou Linux.',
		'Instala la CLI journal en macOS o Linux.',
		'Die journal-CLI auf macOS oder Linux installieren.'
	],
	'downloads.install.script': ['Install script', 'Script d’installation', 'Script de instalación', 'Installationsskript'],
	'downloads.install.facile': ['With the facile CLI', 'Avec la CLI facile', 'Con la CLI facile', 'Mit der facile-CLI'],
	'downloads.install.brew': ['Homebrew', 'Homebrew', 'Homebrew', 'Homebrew']
} as const satisfies Record<string, Row>;
