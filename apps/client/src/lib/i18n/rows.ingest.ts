import type { Row } from './rows';

export const ingest = {
	'alerts.title': ['Alerts', 'Alertes', 'Alertas', 'Warnungen'],
	'alerts.description': [
		'A rule counts matches of a saved query over a rolling window and POSTs to a webhook when the count reaches the threshold.',
		'Une règle compte les correspondances d’une requête enregistrée sur une fenêtre glissante et appelle un webhook (POST) quand le seuil est atteint.',
		'Una regla cuenta las coincidencias de una consulta guardada en una ventana móvil y hace POST a un webhook cuando se alcanza el umbral.',
		'Eine Regel zählt die Treffer einer gespeicherten Abfrage in einem gleitenden Zeitfenster und sendet per POST an einen Webhook, sobald der Schwellenwert erreicht ist.'
	],
	'alerts.newRule': ['New rule', 'Nouvelle règle', 'Nueva regla', 'Neue Regel'],
	'alerts.loadFailed': ['Could not load alert rules', 'Impossible de charger les règles d’alerte', 'No se pudieron cargar las reglas de alerta', 'Warnregeln konnten nicht geladen werden'],
	'alerts.saveQueryFirst': ['Save a query first', 'Enregistrez d’abord une requête', 'Guarde primero una consulta', 'Speichern Sie zuerst eine Abfrage'],
	'alerts.saveQueryHint': [
		'An alert rule watches a named filter set, so there is nothing to watch until one exists.',
		'Une règle d’alerte surveille un jeu de filtres nommé : il n’y a rien à surveiller tant qu’il n’en existe pas.',
		'Una regla de alerta vigila un conjunto de filtros con nombre, así que no hay nada que vigilar hasta que exista uno.',
		'Eine Warnregel überwacht einen benannten Filtersatz; solange keiner existiert, gibt es nichts zu überwachen.'
	],
	'alerts.empty': ['No alert rules yet', 'Aucune règle d’alerte', 'Aún no hay reglas de alerta', 'Noch keine Warnregeln'],
	'alerts.emptyHint': [
		'Pick a saved query, a threshold and a window, and Journal will call your webhook when the count is reached.',
		'Choisissez une requête enregistrée, un seuil et une fenêtre, et Journal appellera votre webhook quand le seuil sera atteint.',
		'Elija una consulta guardada, un umbral y una ventana, y Journal llamará a su webhook cuando se alcance el umbral.',
		'Wählen Sie eine gespeicherte Abfrage, einen Schwellenwert und ein Zeitfenster; Journal ruft Ihren Webhook auf, sobald er erreicht ist.'
	],
	'alerts.rule': ['Rule', 'Règle', 'Regla', 'Regel'],
	'alerts.condition': ['Condition', 'Condition', 'Condición', 'Bedingung'],
	'alerts.conditionValue': ['≥ {threshold} in {minutes}m', '≥ {threshold} en {minutes} min', '≥ {threshold} en {minutes} min', '≥ {threshold} in {minutes} Min.'],
	'alerts.target': ['Target', 'Cible', 'Destino', 'Ziel'],
	'alerts.antenneGlobal': ['Antenne (global)', 'Antenne (global)', 'Antenne (global)', 'Antenne (global)'],
	'alerts.webhook': ['Webhook', 'Webhook', 'Webhook', 'Webhook'],
	'alerts.lastFired': ['Last fired', 'Dernier déclenchement', 'Último disparo', 'Zuletzt ausgelöst'],
	'alerts.enabled': ['Enabled', 'Activée', 'Activada', 'Aktiviert'],
	'alerts.enable': ['Enable {name}', 'Activer {name}', 'Activar {name}', '{name} aktivieren'],
	'alerts.created': ['Alert rule created.', 'Règle d’alerte créée.', 'Regla de alerta creada.', 'Warnregel erstellt.'],
	'alerts.createFailed': ['Could not create the rule', 'Impossible de créer la règle', 'No se pudo crear la regla', 'Regel konnte nicht erstellt werden'],
	'alerts.updateFailed': ['Could not update the rule', 'Impossible de modifier la règle', 'No se pudo actualizar la regla', 'Regel konnte nicht aktualisiert werden'],
	'alerts.deleteFailed': ['Could not delete the rule', 'Impossible de supprimer la règle', 'No se pudo eliminar la regla', 'Regel konnte nicht gelöscht werden'],
	'alerts.deleted': ['Deleted “{name}”.', '« {name} » supprimée.', '«{name}» eliminada.', '„{name}“ gelöscht.'],
	'alerts.drawerTitle': ['New alert rule', 'Nouvelle règle d’alerte', 'Nueva regla de alerta', 'Neue Warnregel'],
	'alerts.drawerHint': [
		'Journal evaluates every rule once a minute and re-arms only after a full window.',
		'Journal évalue chaque règle une fois par minute et ne se réarme qu’après une fenêtre complète.',
		'Journal evalúa cada regla una vez por minuto y solo se rearma tras una ventana completa.',
		'Journal wertet jede Regel einmal pro Minute aus und scharft erst nach einem vollen Zeitfenster wieder.'
	],
	'alerts.createErrorTitle': ['Could not create the rule', 'Impossible de créer la règle', 'No se pudo crear la regla', 'Regel konnte nicht erstellt werden'],
	'alerts.namePlaceholder': ['Nuage error spike', 'Pic d’erreurs Nuage', 'Pico de errores de Nuage', 'Nuage-Fehlerspitze'],
	'alerts.savedQuery': ['Saved query', 'Requête enregistrée', 'Consulta guardada', 'Gespeicherte Abfrage'],
	'alerts.savedQueryHint': [
		'The filter set whose matches are counted.',
		'Le jeu de filtres dont les correspondances sont comptées.',
		'El conjunto de filtros cuyas coincidencias se cuentan.',
		'Der Filtersatz, dessen Treffer gezählt werden.'
	],
	'alerts.pickQuery': ['Pick a saved query…', 'Choisir une requête enregistrée…', 'Elija una consulta guardada…', 'Gespeicherte Abfrage wählen…'],
	'alerts.threshold': ['Threshold', 'Seuil', 'Umbral', 'Schwellenwert'],
	'alerts.thresholdError': ['A whole number, 1 or more.', 'Un nombre entier, 1 ou plus.', 'Un número entero, 1 o más.', 'Eine ganze Zahl, 1 oder mehr.'],
	'alerts.window': ['Window (minutes)', 'Fenêtre (minutes)', 'Ventana (minutos)', 'Zeitfenster (Minuten)'],
	'alerts.windowError': ['Between 1 and 1440.', 'Entre 1 et 1440.', 'Entre 1 y 1440.', 'Zwischen 1 und 1440.'],
	'alerts.webhookUrl': ['Webhook URL', 'URL du webhook', 'URL del webhook', 'Webhook-URL'],
	'alerts.webhookUrlError': ['Must be an http or https URL.', 'Doit être une URL http ou https.', 'Debe ser una URL http o https.', 'Muss eine http- oder https-URL sein.'],
	'alerts.authHeader': ['Auth header', 'En-tête d’authentification', 'Cabecera de autenticación', 'Auth-Header'],
	'alerts.authHeaderHint': [
		'Optional. Name and secret are set together or not at all.',
		'Facultatif. Le nom et le secret vont ensemble ou pas du tout.',
		'Opcional. El nombre y el secreto se definen juntos o ninguno.',
		'Optional. Name und Geheimnis werden zusammen oder gar nicht gesetzt.'
	],
	'alerts.authHeaderError': [
		'Set both the header name and its secret.',
		'Renseignez à la fois le nom de l’en-tête et son secret.',
		'Indique tanto el nombre de la cabecera como su secreto.',
		'Geben Sie sowohl den Header-Namen als auch sein Geheimnis an.'
	],
	'alerts.authSecret': ['Auth secret', 'Secret d’authentification', 'Secreto de autenticación', 'Auth-Geheimnis'],
	'alerts.createRule': ['Create rule', 'Créer la règle', 'Crear regla', 'Regel erstellen'],
	'alerts.deleteTitle': ['Delete this alert rule?', 'Supprimer cette règle d’alerte ?', '¿Eliminar esta regla de alerta?', 'Diese Warnregel löschen?'],
	'alerts.deleteBody': [
		'It stops firing immediately and its webhook is forgotten — nothing will tell you the next time the query matches.',
		'Elle cesse de se déclencher immédiatement et son webhook est oublié : rien ne vous prévient la prochaine fois que la requête correspond.',
		'Deja de dispararse de inmediato y se olvida su webhook: nada le avisará la próxima vez que la consulta coincida.',
		'Sie löst sofort nicht mehr aus und ihr Webhook wird vergessen: Nichts meldet sich, wenn die Abfrage das nächste Mal zutrifft.'
	],
	'keys.endpoints': ['Ingest endpoints', 'Points d’ingestion', 'Puntos de ingesta', 'Ingest-Endpunkte'],
	'keys.endpointsHint': [
		'Where shippers and pages send their entries.',
		'Où les expéditeurs et les pages envoient leurs entrées.',
		'Adonde los emisores y las páginas envían sus entradas.',
		'Wohin Sender und Seiten ihre Einträge schicken.'
	],
	'keys.servers': ['Servers', 'Serveurs', 'Servidores', 'Server'],
	'keys.serversHint': [
		'The trailing /api is load-bearing — without it the dashboard’s catch-all answers 200 and every line is discarded.',
		'Le /api final est indispensable : sans lui, la route fourre-tout du tableau de bord répond 200 et chaque ligne est perdue.',
		'El /api final es imprescindible: sin él, la ruta comodín del panel responde 200 y se descarta cada línea.',
		'Das abschließende /api ist unverzichtbar: Ohne es antwortet der Catch-all des Dashboards mit 200 und jede Zeile geht verloren.'
	],
	'keys.browsers': ['Browsers', 'Navigateurs', 'Navegadores', 'Browser'],
	'keys.browsersHint': [
		'Authenticated by a public key, restricted to that key’s origins, and capped by its daily quota. Used by the @facile/journal SDK.',
		'Authentifié par une clé publique, limité aux origines de cette clé et plafonné par son quota quotidien. Utilisé par le SDK @facile/journal.',
		'Autenticado por una clave pública, restringido a los orígenes de esa clave y limitado por su cuota diaria. Lo usa el SDK @facile/journal.',
		'Authentifiziert durch einen öffentlichen Schlüssel, auf dessen Origins beschränkt und durch sein Tageskontingent begrenzt. Wird vom @facile/journal-SDK verwendet.'
	],
	'keys.title': ['Ingest keys', 'Clés d’ingestion', 'Claves de ingesta', 'Ingest-Schlüssel'],
	'keys.titleHint': [
		'One key is scoped to one app name. Multiple active keys per app are allowed, so rotation is add-new → redeploy → revoke-old with no downtime.',
		'Une clé est liée à un nom d’application. Plusieurs clés actives par application sont permises : la rotation se fait en ajoutant, redéployant puis révoquant l’ancienne, sans interruption.',
		'Cada clave está ligada a un nombre de aplicación. Se permiten varias claves activas por aplicación, así que la rotación es añadir, redesplegar y revocar la antigua, sin interrupción.',
		'Ein Schlüssel gilt für einen App-Namen. Mehrere aktive Schlüssel pro App sind erlaubt; die Rotation läuft also neu anlegen, neu ausrollen, alten widerrufen, ohne Ausfall.'
	],
	'keys.new': ['New key', 'Nouvelle clé', 'Nueva clave', 'Neuer Schlüssel'],
	'keys.loadFailed': ['Could not load API keys', 'Impossible de charger les clés d’API', 'No se pudieron cargar las claves de API', 'API-Schlüssel konnten nicht geladen werden'],
	'keys.empty': ['No ingest keys yet', 'Aucune clé d’ingestion', 'Aún no hay claves de ingesta', 'Noch keine Ingest-Schlüssel'],
	'keys.emptyHint': [
		'Without a key and without the legacy INGEST_TOKEN, every POST to /api/ingest is rejected.',
		'Sans clé et sans l’ancien INGEST_TOKEN, tout POST vers /api/ingest est rejeté.',
		'Sin una clave y sin el antiguo INGEST_TOKEN, todo POST a /api/ingest se rechaza.',
		'Ohne Schlüssel und ohne den alten INGEST_TOKEN wird jeder POST an /api/ingest abgelehnt.'
	],
	'keys.kind': ['Kind', 'Type', 'Tipo', 'Art'],
	'keys.prefix': ['Prefix', 'Préfixe', 'Prefijo', 'Präfix'],
	'keys.scope': ['Scope', 'Portée', 'Alcance', 'Geltungsbereich'],
	'keys.public': ['public', 'publique', 'pública', 'öffentlich'],
	'keys.secret': ['secret', 'secrète', 'secreta', 'geheim'],
	'keys.usedToday': ['{used} / {quota} today', '{used} / {quota} aujourd’hui', '{used} / {quota} hoy', '{used} / {quota} heute'],
	'keys.serversNoQuota': ['servers, no quota', 'serveurs, sans quota', 'servidores, sin cuota', 'Server, kein Kontingent'],
	'keys.revokedOn': ['revoked {date}', 'révoquée le {date}', 'revocada el {date}', 'widerrufen am {date}'],
	'keys.active': ['active', 'active', 'activa', 'aktiv'],
	'keys.revoke': ['Revoke', 'Révoquer', 'Revocar', 'Widerrufen'],
	'keys.revoked': ['Revoked the key for {app}.', 'Clé de {app} révoquée.', 'Clave de {app} revocada.', 'Schlüssel für {app} widerrufen.'],
	'keys.revokeFailed': ['Could not revoke the key', 'Impossible de révoquer la clé', 'No se pudo revocar la clave', 'Schlüssel konnte nicht widerrufen werden'],
	'keys.revokeTitle': ['Revoke this ingest key?', 'Révoquer cette clé d’ingestion ?', '¿Revocar esta clave de ingesta?', 'Diesen Ingest-Schlüssel widerrufen?'],
	'keys.revokeBody': [
		'Any pipeline still using it starts failing immediately, and a revoked key cannot be un-revoked — issue a new one and redeploy instead.',
		'Tout pipeline qui l’utilise encore échoue immédiatement, et une clé révoquée ne peut pas être rétablie : émettez-en une nouvelle et redéployez.',
		'Cualquier flujo que aún la use empieza a fallar de inmediato, y una clave revocada no se puede restaurar: emita una nueva y redespliegue.',
		'Jede Pipeline, die ihn noch nutzt, schlägt sofort fehl, und ein widerrufener Schlüssel lässt sich nicht wiederherstellen: Stellen Sie einen neuen aus und rollen Sie neu aus.'
	],
	'keys.created': ['Key created', 'Clé créée', 'Clave creada', 'Schlüssel erstellt'],
	'keys.newTitle': ['New ingest key', 'Nouvelle clé d’ingestion', 'Nueva clave de ingesta', 'Neuer Ingest-Schlüssel'],
	'keys.newHint': [
		'The app name scopes the key: entries it ships must carry this name or none at all.',
		'Le nom d’application délimite la clé : les entrées envoyées doivent porter ce nom, ou aucun.',
		'El nombre de la aplicación delimita la clave: las entradas que envíe deben llevar este nombre o ninguno.',
		'Der App-Name begrenzt den Schlüssel: Seine Einträge müssen diesen Namen tragen oder gar keinen.'
	],
	'keys.publicAlertTitle': ['This one belongs in your bundle', 'Celle-ci va dans votre bundle', 'Esta va en su bundle', 'Dieser gehört in Ihr Bundle'],
	'keys.publicAlertBody': [
		'A public key is meant to be shipped to browsers. What protects Journal is the origin allowlist, the rate limit and the daily quota — not the secrecy of this string.',
		'Une clé publique est faite pour être livrée aux navigateurs. Ce qui protège Journal, c’est la liste d’origines autorisées, la limite de débit et le quota quotidien, pas le secret de cette chaîne.',
		'Una clave pública está pensada para enviarse a los navegadores. Lo que protege a Journal es la lista de orígenes permitidos, el límite de frecuencia y la cuota diaria, no el secreto de esta cadena.',
		'Ein öffentlicher Schlüssel ist dafür gedacht, an Browser ausgeliefert zu werden. Journal schützen die Origin-Allowlist, das Ratenlimit und das Tageskontingent, nicht die Geheimhaltung dieser Zeichenfolge.'
	],
	'keys.copyNowTitle': ['Copy it now', 'Copiez-la maintenant', 'Cópiela ahora', 'Jetzt kopieren'],
	'keys.copyNowBody': [
		'This is the only time the token is shown. Journal stores nothing but its SHA-256.',
		'Le jeton n’est affiché qu’une seule fois. Journal ne conserve que son SHA-256.',
		'El token se muestra solo esta vez. Journal solo guarda su SHA-256.',
		'Das Token wird nur dieses eine Mal angezeigt. Journal speichert nur seinen SHA-256.'
	],
	'keys.tokenFor': ['Token for {app}', 'Jeton pour {app}', 'Token para {app}', 'Token für {app}'],
	'keys.createErrorTitle': ['Could not create the key', 'Impossible de créer la clé', 'No se pudo crear la clave', 'Schlüssel konnte nicht erstellt werden'],
	'keys.checkFields': [
		'Check the app name, the origins and the quota.',
		'Vérifiez le nom de l’application, les origines et le quota.',
		'Compruebe el nombre de la aplicación, los orígenes y la cuota.',
		'Prüfen Sie App-Name, Origins und Kontingent.'
	],
	'keys.optionSecret': ['Secret — servers', 'Secrète — serveurs', 'Secreta — servidores', 'Geheim — Server'],
	'keys.optionPublic': ['Public — browsers', 'Publique — navigateurs', 'Pública — navegadores', 'Öffentlich — Browser'],
	'keys.appName': ['App name', 'Nom de l’application', 'Nombre de la aplicación', 'App-Name'],
	'keys.appNameHint': [
		'Lowercase letters, digits and hyphens.',
		'Lettres minuscules, chiffres et tirets.',
		'Letras minúsculas, dígitos y guiones.',
		'Kleinbuchstaben, Ziffern und Bindestriche.'
	],
	'keys.appNameError': [
		'Must match ^[a-z0-9][a-z0-9-]{0,63}$.',
		'Doit respecter ^[a-z0-9][a-z0-9-]{0,63}$.',
		'Debe cumplir ^[a-z0-9][a-z0-9-]{0,63}$.',
		'Muss ^[a-z0-9][a-z0-9-]{0,63}$ entsprechen.'
	],
	'keys.origins': ['Allowed origins', 'Origines autorisées', 'Orígenes permitidos', 'Erlaubte Origins'],
	'keys.originsHint': [
		'One per line, scheme://host[:port], no path and no wildcard. Up to 8. Anything else is refused.',
		'Une par ligne, schéma://hôte[:port], sans chemin ni joker. 8 au maximum. Tout le reste est refusé.',
		'Una por línea, esquema://host[:puerto], sin ruta ni comodín. Hasta 8. Todo lo demás se rechaza.',
		'Eine pro Zeile, Schema://Host[:Port], ohne Pfad und ohne Platzhalter. Höchstens 8. Alles andere wird abgelehnt.'
	],
	'keys.originsError': [
		'Each line must look like https://shop.example or http://localhost:5173, 8 at most.',
		'Chaque ligne doit ressembler à https://boutique.example ou http://localhost:5173, 8 au maximum.',
		'Cada línea debe parecerse a https://tienda.example o http://localhost:5173, 8 como máximo.',
		'Jede Zeile muss wie https://shop.example oder http://localhost:5173 aussehen, höchstens 8.'
	],
	'keys.quota': ['Daily quota', 'Quota quotidien', 'Cuota diaria', 'Tageskontingent'],
	'keys.quotaHint': [
		'Entries accepted per UTC day. The bound that holds if this key is abused — set it to a few times what a normal day looks like.',
		'Entrées acceptées par jour UTC. C’est la limite qui tient si la clé est détournée : fixez-la à quelques fois une journée normale.',
		'Entradas aceptadas por día UTC. Es el límite que se mantiene si se abusa de esta clave: fíjelo en unas cuantas veces un día normal.',
		'Pro UTC-Tag akzeptierte Einträge. Diese Grenze hält, falls der Schlüssel missbraucht wird: Setzen Sie sie auf ein Mehrfaches eines normalen Tages.'
	],
	'keys.quotaError': ['Between 1 and 10000000.', 'Entre 1 et 10000000.', 'Entre 1 y 10000000.', 'Zwischen 1 und 10000000.'],
	'keys.createKey': ['Create key', 'Créer la clé', 'Crear clave', 'Schlüssel erstellen'],
	'antenne.title': ['Antenne', 'Antenne', 'Antenne', 'Antenne'],
	'antenne.hint': [
		'Configure the global Antenne instance for alert delivery.',
		'Configurez l’instance Antenne globale pour l’envoi des alertes.',
		'Configure la instancia global de Antenne para el envío de alertas.',
		'Konfigurieren Sie die globale Antenne-Instanz für die Zustellung von Warnungen.'
	],
	'antenne.loadFailed': ['Could not load Antenne settings', 'Impossible de charger les réglages Antenne', 'No se pudieron cargar los ajustes de Antenne', 'Antenne-Einstellungen konnten nicht geladen werden'],
	'antenne.enabled': ['Enabled', 'Activé', 'Activado', 'Aktiviert'],
	'antenne.enabledHint': [
		'When enabled, alerts will be sent to the configured Antenne instance.',
		'Une fois activé, les alertes sont envoyées à l’instance Antenne configurée.',
		'Si se activa, las alertas se envían a la instancia de Antenne configurada.',
		'Wenn aktiviert, werden Warnungen an die konfigurierte Antenne-Instanz gesendet.'
	],
	'antenne.url': ['Instance URL', 'URL de l’instance', 'URL de la instancia', 'Instanz-URL'],
	'antenne.urlHint': [
		'The URL of your Antenne instance (e.g., https://antenne.facile.studio)',
		'L’URL de votre instance Antenne (par ex. https://antenne.facile.studio)',
		'La URL de su instancia de Antenne (p. ej., https://antenne.facile.studio)',
		'Die URL Ihrer Antenne-Instanz (z. B. https://antenne.facile.studio)'
	],
	'antenne.secret': ['Secret', 'Secret', 'Secreto', 'Geheimnis'],
	'antenne.secretHint': [
		'The secret key for authenticating with Antenne',
		'La clé secrète pour s’authentifier auprès d’Antenne',
		'La clave secreta para autenticarse en Antenne',
		'Der geheime Schlüssel für die Authentifizierung bei Antenne'
	],
	'antenne.secretLabel': ['Antenne Secret', 'Secret Antenne', 'Secreto de Antenne', 'Antenne-Geheimnis'],
	'antenne.connected': ['Connected', 'Connecté', 'Conectado', 'Verbunden'],
	'antenne.error': ['Error: {message}', 'Erreur : {message}', 'Error: {message}', 'Fehler: {message}'],
	'antenne.disconnected': ['Disconnected', 'Déconnecté', 'Desconectado', 'Getrennt'],
	'antenne.updated': ['Antenne settings updated.', 'Réglages Antenne mis à jour.', 'Ajustes de Antenne actualizados.', 'Antenne-Einstellungen aktualisiert.'],
	'antenne.updateFailed': ['Failed to update Antenne settings', 'Impossible de mettre à jour les réglages Antenne', 'No se pudieron actualizar los ajustes de Antenne', 'Antenne-Einstellungen konnten nicht aktualisiert werden'],
	'keys.loadSettingsFailed': ['Failed to load settings', 'Impossible de charger les réglages', 'No se pudieron cargar los ajustes', 'Einstellungen konnten nicht geladen werden']
} as const satisfies Record<string, Row>;
