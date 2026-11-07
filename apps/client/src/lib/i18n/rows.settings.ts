import type { Row } from './rows';

export const settings = {
	'settings.title': ['Settings', 'Réglages', 'Ajustes', 'Einstellungen'],
	'settings.sections': ['Settings sections', 'Sections des réglages', 'Secciones de ajustes', 'Einstellungsbereiche'],
	'settings.profile': ['Profile', 'Profil', 'Perfil', 'Profil'],
	'settings.preferences': ['Preferences', 'Préférences', 'Preferencias', 'Präferenzen'],
	'settings.api': ['API', 'API', 'API', 'API'],
	'settings.advanced': ['Advanced', 'Avancé', 'Avanzado', 'Erweitert'],
	'profile.admin': ['Admin', 'Administrateur', 'Administrador', 'Admin'],
	'profile.member': ['Member', 'Membre', 'Miembro', 'Mitglied'],
	'profile.since': ['Member since', 'Membre depuis', 'Miembro desde', 'Mitglied seit'],
	'profile.danger': ['Danger zone', 'Zone de danger', 'Zona de peligro', 'Gefahrenbereich'],
	'profile.dangerHint': [
		'Erasure under GDPR Article 17. The account and every session are destroyed. Log entries stay until they expire (RETENTION_DAYS): they name apps, and an app can write personal data into a message.',
		'Effacement au titre de l’article 17 du RGPD. Le compte et toutes les sessions sont détruits. Les entrées de journal restent jusqu’à leur expiration (RETENTION_DAYS) : elles nomment des applications, et une application peut écrire des données personnelles dans un message.',
		'Supresión según el artículo 17 del RGPD. La cuenta y todas las sesiones se destruyen. Las entradas de registro permanecen hasta su caducidad (RETENTION_DAYS): nombran aplicaciones y una aplicación puede escribir datos personales en un mensaje.',
		'Löschung nach Artikel 17 DSGVO. Das Konto und alle Sitzungen werden zerstört. Log-Einträge bleiben bis zu ihrem Ablauf (RETENTION_DAYS): Sie nennen Apps, und eine App kann personenbezogene Daten in eine Nachricht schreiben.'
	],
	'profile.delete': ['Delete this account', 'Supprimer ce compte', 'Eliminar esta cuenta', 'Dieses Konto löschen'],
	'profile.deleteHint': [
		'Permanent. The last administrator cannot delete the account until another admin exists.',
		'Définitif. Le dernier administrateur ne peut pas supprimer son compte tant qu’un autre administrateur n’existe pas.',
		'Permanente. El último administrador no puede eliminar la cuenta hasta que exista otro administrador.',
		'Endgültig. Der letzte Administrator kann das Konto erst löschen, wenn es einen weiteren Admin gibt.'
	],
	'profile.deleteButton': ['Delete account…', 'Supprimer le compte…', 'Eliminar la cuenta…', 'Konto löschen…'],
	'profile.deleteFailed': ['Could not delete the account', 'Impossible de supprimer le compte', 'No se pudo eliminar la cuenta', 'Konto konnte nicht gelöscht werden'],
	'profile.confirmTitle': ['Delete your account?', 'Supprimer votre compte ?', '¿Eliminar su cuenta?', 'Ihr Konto löschen?'],
	'profile.confirmBody': [
		'You are signed out everywhere immediately, and there is no way back. Log entries already stored stay until they expire.',
		'Vous êtes déconnecté partout immédiatement, sans retour possible. Les entrées de journal déjà stockées restent jusqu’à leur expiration.',
		'Se cierra su sesión en todas partes de inmediato, sin vuelta atrás. Las entradas de registro ya almacenadas permanecen hasta su caducidad.',
		'Sie werden sofort überall abgemeldet, ein Zurück gibt es nicht. Bereits gespeicherte Log-Einträge bleiben bis zu ihrem Ablauf.'
	],
	'profile.confirmButton': ['Delete forever', 'Supprimer définitivement', 'Eliminar para siempre', 'Endgültig löschen'],
	'profile.deleting': ['Deleting…', 'Suppression…', 'Eliminando…', 'Wird gelöscht…'],
	'advanced.instance': ['Instance', 'Instance', 'Instancia', 'Instanz'],
	'advanced.instanceHint': [
		'Facts about the server this dashboard is talking to.',
		'Informations sur le serveur auquel ce tableau de bord parle.',
		'Datos del servidor con el que habla este panel.',
		'Angaben zum Server, mit dem dieses Dashboard spricht.'
	],
	'advanced.apiBase': ['API base', 'Base de l’API', 'Base de la API', 'API-Basis'],
	'advanced.statusHint': [
		'Whether the dashboard can reach the API right now.',
		'Indique si le tableau de bord joint l’API en ce moment.',
		'Indica si el panel alcanza la API en este momento.',
		'Zeigt, ob das Dashboard die API gerade erreicht.'
	],
	'advanced.checking': ['Checking…', 'Vérification…', 'Comprobando…', 'Wird geprüft…'],
	'advanced.reachable': ['Reachable', 'Joignable', 'Accesible', 'Erreichbar'],
	'advanced.unreachable': ['Unreachable', 'Injoignable', 'Inaccesible', 'Nicht erreichbar'],
	'advanced.signIn': ['Sign-in', 'Connexion', 'Acceso', 'Anmeldung'],
	'advanced.signInHint': [
		'Accounts are created in Registre. Journal has no sign-up and no password of its own.',
		'Les comptes se créent dans Registre. Journal n’a ni inscription ni mot de passe propre.',
		'Las cuentas se crean en Registre. Journal no tiene registro ni contraseña propios.',
		'Konten werden in Registre angelegt. Journal hat weder eine eigene Registrierung noch ein eigenes Passwort.'
	],
	'advanced.signInValue': ['Facile account', 'Compte Facile', 'Cuenta Facile', 'Facile-Konto'],
	'advanced.retention': ['Retention', 'Conservation', 'Retención', 'Aufbewahrung'],
	'advanced.retentionHint': [
		'Entries older than RETENTION_DAYS are deleted by an hourly job on the server; 0 keeps them forever. It is a deploy-time setting, not a dashboard one.',
		'Les entrées plus anciennes que RETENTION_DAYS sont supprimées par une tâche horaire sur le serveur ; 0 les conserve indéfiniment. C’est un réglage de déploiement, pas du tableau de bord.',
		'Las entradas más antiguas que RETENTION_DAYS las elimina una tarea horaria en el servidor; 0 las conserva para siempre. Es un ajuste de despliegue, no del panel.',
		'Einträge, die älter als RETENTION_DAYS sind, löscht ein stündlicher Job auf dem Server; 0 behält sie für immer. Das ist eine Einstellung beim Deployment, keine des Dashboards.'
	],
	'advanced.where': ['Where to change it', 'Où la modifier', 'Dónde cambiarlo', 'Wo es geändert wird'],
	'advanced.whereHint': [
		'Dokploy environment for this stack, alongside DATABASE_URL and CORS_ALLOWED_ORIGINS.',
		'Environnement Dokploy de cette pile, avec DATABASE_URL et CORS_ALLOWED_ORIGINS.',
		'Entorno de Dokploy de este stack, junto a DATABASE_URL y CORS_ALLOWED_ORIGINS.',
		'Dokploy-Umgebung dieses Stacks, neben DATABASE_URL und CORS_ALLOWED_ORIGINS.'
	]
} as const satisfies Record<string, Row>;
