import type { Row } from '@facile/muse';

export const landingBlocks = {
	'landing.with.line': [
		'Journal hands its alerts to the suite’s alert bus.',
		'Journal confie ses alertes au bus d’alertes de la suite.',
		'Journal entrega sus alertas al bus de alertas de la suite.',
		'Journal übergibt seine Alarme dem Alarmbus der Suite.'
	],
	'landing.with.antenne': [
		'Your alerts go out through Antenne.',
		'Vos alertes partent par Antenne.',
		'Tus alertas salen por Antenne.',
		'Ihre Alarme laufen über Antenne.'
	],
	'landing.pricing.title': ['Plans', 'Les offres', 'Los planes', 'Die Tarife'],
	'landing.pricing.free': [
		'10,000 entries a day, kept 7 days',
		'10 000 entrées par jour, gardées 7 jours',
		'10.000 entradas al día, guardadas 7 días',
		'10.000 Einträge pro Tag, 7 Tage aufbewahrt'
	],
	'landing.pricing.freeSoon': [
		'Soon: 10,000 entries a day, kept 7 days',
		'Bientôt : 10 000 entrées par jour, gardées 7 jours',
		'Pronto: 10.000 entradas al día, guardadas 7 días',
		'Bald: 10.000 Einträge pro Tag, 7 Tage aufbewahrt'
	],
	'landing.pricing.included': [
		'Journal included, with every other app',
		'Journal inclus, avec toutes les autres apps',
		'Journal incluido, con todas las demás apps',
		'Journal inklusive, mit allen anderen Apps'
	],
	'landing.pricing.soon': [
		'Journal opens to every Facile account soon',
		'Journal ouvre bientôt à tous les comptes Facile',
		'Journal se abre pronto a todas las cuentas Facile',
		'Journal öffnet bald für alle Facile-Konten'
	],
	'landing.pricing.suite': [
		'Journal is part of the Facile suite: one subscription for every app.',
		'Journal fait partie de la suite Facile : un seul abonnement pour toutes les apps.',
		'Journal forma parte de la suite Facile: una suscripción para todas las apps.',
		'Journal gehört zur Facile-Suite: ein Abo für alle Apps.'
	],
	'landing.dev.line': [
		'Go and browser SDKs, a Docker collector, and a CLI to follow your logs live. Open source, and you can run it yourself: see self-hosting.',
		'SDK Go et navigateur, collecteur Docker, et un CLI pour suivre vos logs en direct. Open source, installable chez vous : voir l’auto-hébergement.',
		'SDK para Go y navegador, colector de Docker y un CLI para seguir tus logs en directo. Código abierto, y puedes instalarlo tú mismo: ver autoalojamiento.',
		'SDKs für Go und Browser, ein Docker-Collector und eine CLI, um Logs live zu verfolgen. Open Source, auch selbst betreibbar: siehe Self-Hosting.'
	],
	'landing.dev.cli': [
		'journal tail follows new lines and journal logs searches them, even over SSH.',
		'journal tail suit les nouvelles lignes et journal logs les cherche, même en SSH.',
		'journal tail sigue las líneas nuevas y journal logs las busca, incluso por SSH.',
		'journal tail folgt neuen Zeilen, journal logs durchsucht sie, auch über SSH.'
	],
	'landing.dev.api': [
		'Send entries with one POST to /api/ingest, from any language.',
		'Envoyez vos entrées en un POST sur /api/ingest, depuis n’importe quel langage.',
		'Envía tus entradas con un POST a /api/ingest, desde cualquier lenguaje.',
		'Senden Sie Einträge mit einem POST an /api/ingest, aus jeder Sprache.'
	],
	'landing.dev.mcp': [
		'Let your AI assistant search your logs, with your account’s access.',
		'Laissez votre assistant IA chercher dans vos logs, avec les droits de votre compte.',
		'Deja que tu asistente de IA busque en tus logs, con los permisos de tu cuenta.',
		'Lassen Sie Ihren KI-Assistenten Ihre Logs durchsuchen, mit den Rechten Ihres Kontos.'
	],
	'landing.dev.source': [
		'The API, dashboard, collector and SDKs, on GitHub.',
		'L’API, le tableau de bord, le collecteur et les SDK, sur GitHub.',
		'La API, el panel, el colector y los SDK, en GitHub.',
		'API, Dashboard, Collector und SDKs, auf GitHub.'
	],
	'landing.cta.title': ['Start reading your logs', 'Lisez vos logs', 'Empieza a leer tus logs', 'Logs jetzt lesen'],
	'landing.cta.open': [
		'Create a free Facile account, or sign in with yours.',
		'Créez un compte Facile gratuit, ou connectez-vous avec le vôtre.',
		'Crea una cuenta Facile gratuita, o inicia sesión con la tuya.',
		'Erstellen Sie ein kostenloses Facile-Konto oder melden Sie sich mit Ihrem an.'
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
	'downloads.install.kit': ['With kit', 'Avec kit', 'Con kit', 'Mit kit'],
	'downloads.install.brew': ['Homebrew', 'Homebrew', 'Homebrew', 'Homebrew']
} as const satisfies Record<string, Row>;
