# Sources de données

Inventaire contraignant : une source absente de ce fichier n'est pas utilisée. Relevés du 29/10/2026 (étude de faisabilité).

## Synchronisation

`cmd/sync` télécharge chaque liste à son rythme (`internal/adapter/source/catalog`), plus un délai aléatoire d'au plus un dixième de l'intervalle : toutes les 6 heures les listes AMF (fraîcheur quotidienne) et la liste ACPR (fraîcheur non annoncée), toutes les 24 heures la numérotation ARCEP (fraîcheur hebdomadaire). Au démarrage, une source dont l'instantané stocké a moins d'un intervalle attend son tour. Un échec ou un rejet garde l'instantané précédent et lève une alerte (journal `error` et webhook `ARNAKOFF_ALERT_WEBHOOK`). `cmd/api` recharge les instantanés stockés chaque minute, sans redémarrage.

## Description dans `GET /v1/sources`

Chaque source y est décrite par son éditeur, sa licence et sa page publique, déclarés à côté de son connecteur ou de son registre (`info.go`). Les licences AMF, ARCEP et BODACC sont celles de leurs jeux data.gouv.fr (relevées le 02/11/2026) ; celle de SIRENE vient de la section SIRENE ci-dessous. Sans licence publiée, quatre libellés ont été proposés puis acceptés par le propriétaire (02/11/2026) :

| Source | Éditeur | Licence | Page |
|---|---|---|---|
| ACPR | ACPR et AMF (ABE Info Service) | Autorisation de l'éditeur | `https://www.abe-infoservice.fr/liste-noire` |
| RDAP | Registres des extensions (amorçage IANA) | Données publiques d'enregistrement | `https://www.iana.org/assignments/rdap-dns` |
| IBAN | SWIFT (registre IBAN, norme ISO 13616) | Aucune donnée téléchargée | page du registre IBAN de SWIFT |
| Domaines officiels | Arnakoff | Aucune licence tierce | `domains.csv` sur GitHub |

L'ACPR n'est listée qu'une fois la source activée, donc après l'autorisation de réutilisation (#30). L'IBAN reste listé bien qu'aucune donnée ne soit téléchargée.

## Liste noire AMF

- **Éditeur** : Autorité des marchés financiers.
- **Licence** : Licence Ouverte 2.0 (`lov2`), réutilisation commerciale autorisée.
- **Accès** : fichier CSV sur data.gouv.fr, jeu `641d5dab1d84b530f7720b08`. URL du fichier horodatée (`export_od_listesnoires_<début>_<fin>.csv`) : la résoudre via `https://www.data.gouv.fr/api/1/datasets/641d5dab1d84b530f7720b08/`.
- **Format** : UTF-8 avec BOM, champs entre guillemets, séparateur `;`, fin de ligne `LF`, colonnes `nom;categorie;date_inscription` (date ISO). Catégories multiples séparées par `|` (76 lignes), libellés avec `_` (`Produits_dérivés_sur_crypto-actifs`, `Biens_divers/Or`). Le fichier est servi par `object-api.infra.data.gouv.fr`, environ 190 Ko.
- **Contenu de `nom`** : un site (`armen.ltd`, parfois avec `https://`, 146 avec un chemin vers une page précise comme `calendly.com/amf-france`), une adresse e-mail (souvent générique, `prénom.nom@<domaine>`), ou un nom libre. Quelques fautes de saisie : point final (`prénom.nom@lgt-bp.com.`, corrigé), extension inexistante (`support@amf-france.tram`, écartée). Des entrées figurent plusieurs fois avec des dates différentes ; chaque inscription est gardée.
- **Fraîcheur** : quotidienne annoncée et observée.
- **Référence** (29/10/2026) : 3 499 lignes (2 550 domaines, 919 e-mails, 30 noms), dernière inscription le 08/10/2026 ; 3 466 lignes retenues dans l'instantané. 350 à 430 inscriptions par an.
- **Données personnelles** : oui pour une partie des noms libres, qui citent des personnes physiques. Décision du propriétaire (29/10/2026) : les noms libres sont écartés de l'instantané et des fixtures ; seuls les sites et les adresses e-mail sont gardés.

## Listes blanches AMF

Deux sources distinctes, chacune avec son instantané, son minimum de lignes et son contrôle de baisse de 20 % (décision du propriétaire, 29/10/2026) : « AMF PSAN » et « AMF biens divers ».

- **Éditeur** : AMF. **Licence** : Licence Ouverte 2.0 (`lov2`), réutilisation commerciale autorisée.
- **Accès** : data.gouv.fr, un seul fichier CSV `main` par jeu, URL horodatée à résoudre via l'API comme pour la liste noire : jeux `653135cf052c9a87787413c2` (PSAN) et `65709c9cae162fbc34e7ec05` (biens divers). Fichiers servis par `object-api.infra.data.gouv.fr`.
- **Format** : UTF-8 avec BOM, champs entre guillemets, séparateur `;`, fin de ligne `LF`, dates ISO.
  - PSAN : `no_amf;entite_nom;forme_juridique;pays_siege;site_internet;email;telephone;no_registre_national;lei;nature_autorisation;date_debut_autorisation;date_fin_autorisation;motif_fin_autorisation;statut;libelle_type_activite;libelle_activite;date_debut_activite;date_fin_activite;date_de_publication`. Une ligne par activité d'un prestataire. `statut` vaut `Agréé` ou `Radié` ; `nature_autorisation` `Enregistrement`, `Agrément MICA` ou `Agrément Pacte` ; `date_fin_autorisation` vide tant que l'autorisation court. `forme_juridique` est toujours vide.
  - Biens divers : `no_amf;offre_nom;url_offre;telephone;date_enregistrement;date_de_publication`. Une ligne par offre ; `url_offre` et `telephone` souvent vides.
- **Contenu PSAN** : `no_registre_national` est un SIREN pour les prestataires français (135, tous valides) et un numéro de registre étranger sinon ; les prestataires passeportés depuis un autre pays de l'UE partagent le numéro `N/A - Passeport` (164). La colonne `email` contient parfois l'URL d'un formulaire de contact (10 prestataires), `site_internet` parfois deux sites séparés par une espace, une fois une faute (`ttps://`, corrigée). Un champ `telephone` contient deux numéros et est écarté ; un autre finit par une espace insécable.
- **Fraîcheur** : quotidienne annoncée ; dernière autorisation PSAN observée le 23/10/2026, dernière offre enregistrée le 27/01/2026.
- **Référence** (29/10/2026) : PSAN 1 168 lignes, 158 numéros AMF (37 agréés, 121 radiés), 607 lignes retenues (314 sites, 101 adresses, 57 numéros, 135 SIREN) ; biens divers 12 offres, 12 lignes retenues (6 sites, 6 numéros).
- **Données personnelles** : non, les entrées sont des personnes morales.
- **Usage** : signaux `info` `amf_registered` et `amf_registration_ended` (`docs/signals.md`), jamais une garantie ; un prestataire `Radié` reste listé avec sa date de fin. Une ligne PSAN `Radié` sans date de fin rejette le fichier : l'instantané précédent reste en service.

## Liste noire ACPR (ABE Info Service)

- **Éditeur** : ACPR et AMF, via Assurance Banque Épargne Info Service.
- **Licence** : **non indiquée, à obtenir avant tout usage commercial** (issue #30, réservée au propriétaire). Le connecteur existe mais la source reste désactivée par défaut (`ARNAKOFF_SOURCE_ACPR=false`) tant que #30 n'est pas fermée.
- **Accès** : `https://www.abe-infoservice.fr/fr/abeis-liste-noire.csv`, réponse directe sans redirection, `Content-Type: text/csv; charset=UTF-8`, environ 970 Ko.
- **Format** : UTF-8 avec BOM, séparateur `,`, fin de ligne `LF`, guillemets seulement si nécessaire, colonnes `"URL / Mails",Dénomination,Catégorie(s),"Date d'inscription"` (date `JJ/MM/AAAA`), catégories multiples séparées par `, ` (`Crédits livrets paiements assurances, Usurpation Professionnel`, `Biens divers : or`). Dans l'instantané, les catégories sont séparées par `|` et la date est au format ISO, comme pour la liste noire AMF.
- **Contenu de `URL / Mails`** (relevé du 30/10/2026) : 8 274 sites (sans `https://`, 343 avec un chemin vers une page précise), 3 226 adresses e-mail, 31 valeurs avec une espace (noms libres, dont des noms de personnes physiques, et fautes comme `trader-ai.fr ok` ou `prénom.nom@ inversis-invest.com`), écartées. 45 valeurs finissent par une espace, parfois insécable, retirée. 2 entrées sans catégorie (`stabletrader-ai.net`, `monplantfraise.fr`) sont gardées sans catégorie ; des entrées figurent plusieurs fois et chaque inscription est gardée.
- **Dénomination** : renseignée pour 3 252 lignes, souvent le nom d'une société ; elle peut nommer une personne physique. Elle n'est pas reprise dans l'instantané, par cohérence avec la décision du propriétaire sur les noms libres de la liste noire AMF.
- **Fraîcheur** : non annoncée ; en-tête `Last-Modified` du jour au 30/10/2026, dernière inscription observée le 26/10/2026, inscriptions depuis le 01/12/2010.
- **Référence** (30/10/2026) : 11 531 lignes, 11 455 retenues dans l'instantané (11 421 clés distinctes). Elle reprend des entrées AMF : 1 917 valeurs identiques au 29/10/2026, 1 936 clés communes une fois normalisées de la même façon (hôte, chemin, adresse). Union des clés AMF et ACPR : 12 947.
- **Dédoublonnage** : les deux connecteurs normalisent les entrées avec le même code (`internal/adapter/source/listing`), si bien qu'une entrée commune a la même clé dans les deux instantanés. Chaque instantané garde ses lignes ; une entrée trouvée dans les deux listes donne un signal par liste, `amf_blacklist` et `acpr_blacklist`, chacun citant sa source (`docs/signals.md`).
- **Corpus d'évaluation** : 160 entrées de `testdata/corpus/scam.csv` viennent de cette liste (75 aussi sur la liste noire AMF). Décision du propriétaire (30/10/2026) : la liste est incluse dans le corpus sans attendre la réponse sur la licence (#30).
- **Données personnelles** : oui pour les noms libres et une partie des dénominations, écartés ; les adresses e-mail retenues sont souvent génériques (`prénom.nom@<domaine>`).

## SIRENE (API Recherche d'entreprises)

- **Éditeur** : DINUM, données INSEE. **Licence** : Licence Ouverte 2.0.
- **Accès** : `https://recherche-entreprises.api.gouv.fr/search?q=<siren>&minimal=true&include=siege&per_page=5`, sans clé. `minimal=true&include=siege` exclut les dirigeants (données personnelles), qui ne sont donc jamais téléchargés. La recherche est textuelle : seul le résultat dont le `siren` est celui demandé est retenu.
- **Quotas** (documentation officielle, `https://recherche-entreprises.api.gouv.fr/openapi.json`, relevée le 30/10/2026 et le 06/11/2026) : au plus 7 requêtes par seconde par adresse IP et 30 par ASN ; au-delà, réponse 429 avec `Retry-After`. Scaleway ne garantit pas l'adresse IP de sortie de ses conteneurs (FAQ Serverless Containers) : plusieurs instances peuvent en partager une. Chaque instance s'en tient donc à 7 requêtes par seconde divisées par `ARNAKOFF_INSTANCES`, le nombre maximal d'instances (#336), en une seule tentative par recherche, sans nouvelle tentative qui échapperait à cette limite (#343), et respecte `Retry-After`. La limite par ASN est partagée avec les autres clients de l'hébergeur : seul `Retry-After` la protège. Les entreprises non diffusibles ne sont pas accessibles par cette API.
- **Champs utilisés** : `siren`, `nom_complet`, `date_creation`, `siege.etat_administratif` (`A` active, `F` fermée), `siege.date_fermeture`.

## BODACC

- **Éditeur** : DILA. **Licence** : Licence Ouverte.
- **Accès** : `https://bodacc-datadila.opendatasoft.com/api/explore/v2.1/catalog/datasets/annonces-commerciales/records`, sans clé, filtre `registre="<siren>" and familleavis in ("collective","conciliation","radiation","retablissement_professionnel")`, `order_by=dateparution desc`, `limit=100` (le maximum accepté : `limit=101` est refusé). `registre` est une liste qui porte le SIREN avec et sans espaces, et les SIREN de toutes les sociétés d'un même avis ; l'égalité porte sur un élément.
- **Données personnelles** : un avis nomme des personnes (`commercant`, `listepersonnes` : nom, prénom, adresse). `select=id,registre,dateparution,familleavis,typeavis,publicationavis,parution,numeroannonce,parutionavisprecedent,url_complete` ne demande aucun de ces champs, qui ne sont donc jamais téléchargés.
- **Familles d'avis** (`familleavis`, relevées par facette le 30/10/2026) : `dpc` (Dépôts des comptes), `modification`, `creation`, `radiation` (Radiations), `collective` (Procédures collectives), `vente`, `immatriculation`, `divers`, `conciliation` (Procédures de conciliation), `retablissement_professionnel` (Procédures de rétablissement professionnel), `inconnue`. Seules les quatre familles nommées sont lues, chacune gardée distincte (`domain.NoticeKind`) ; ce qu'elles signifient pour un signal est décidé par la règle (#44).
- **Avis rectificatifs et d'annulation** (`typeavis` : `annonce`, `rectificatif`, `annulation`) : `parutionavisprecedent` désigne l'avis visé par `nomPublication` (« BODACC A »), `numeroParution` et `numeroAnnonce`, qui sont le `publicationavis`, la `parution` et le `numeroannonce` de cet avis (vérifié sur le SIREN 835084732 : l'avis A202500814208 annule A202500322546). Un avis annulé est écarté avec son annulation ; un avis rectifié cède la place à son rectificatif.
- **Quotas** : aucun quota par seconde publié ; les réponses portent `X-RateLimit-Limit: 5000000` et `X-RateLimit-Reset` au jour suivant (relevé le 30/10/2026). Réponses en cache 24 h (le BODACC paraît une fois par jour), `Retry-After` respecté.
- **Limites** : un avis sur une personne inscrite au seul répertoire des métiers a `registre` vide (son numéro n'est que dans `listepersonnes`) et n'est pas trouvé par SIREN.
- **Champs utilisés** : `id`, `registre`, `dateparution`, `familleavis`, `typeavis`, `publicationavis`, `parution`, `numeroannonce`, `parutionavisprecedent`, `url_complete` (page publique de l'avis, `https://www.bodacc.fr/pages/annonces-commerciales-detail/?q.id=id:<id>`, vérifiée le 02/11/2026 sur les 13 avis des fixtures ; une autre adresse rend la réponse invalide).

## RDAP (âge des domaines)

- **Éditeurs** : le registre de chaque extension, trouvé par le fichier d'amorçage IANA (RFC 9224) `https://data.iana.org/rdap/dns.json` (200, `Cache-Control: max-age=86400`, 1 203 extensions et 379 serveurs le 30/10/2026), relu toutes les 24 h. Exemples : AFNIC pour `.fr` (`https://rdap.nic.fr/domain/<d>`), Verisign pour `.com` (`https://rdap.verisign.com/com/v1/domain/<d>`). Jamais `rdap.org`. Seules les URL HTTPS sont retenues (`.kg` et `.mg` n'en ont pas) ; certaines extensions (`.io` par exemple) n'ont aucun serveur dans le fichier : pas de signal. La liste d'hôtes autorisés du client RDAP est celle du fichier.
- **Licence** : données d'enregistrement publiques ; seuls les événements (`registration`, `last changed`) sont conservés, jamais les contacts.
- **Champs utilisés** : `events[].eventAction == "registration"`. `eventDate` est une date-heure RFC 3339 (RFC 9083, section 4.5) ; une date seule, publiée par le registre de `.si` (`rdap.register.si`, relevé le 30/10/2026 pour `ilirika.si` : `"1997-09-02"`), est lue à minuit UTC. Aucun autre registre du corpus d'évaluation ne s'en écarte (478 cas, 30/10/2026).
- **Limites** : quotas par registre (`429` observé sur `rdap.org`) : réponses en cache 24 h, « introuvable » (`404`) 5 min, et un registre qui renvoie `Retry-After` n'est plus interrogé pendant ce délai (une heure au plus).

## Numérotation ARCEP (MAJNUM)

Une source, « ARCEP » : le fichier des tranches attribuées, joint au fichier des opérateurs pour nommer l'attributaire de chaque tranche.

- **Éditeur** : Autorité de régulation des communications électroniques, des postes et de la distribution de la presse (ARCEP).
- **Licence** : Licence Ouverte (`fr-lo`, version 1.0), réutilisation commerciale autorisée avec mention de la source, relevée sur les jeux data.gouv.fr `575067fe88ee382bfa84e966` (« Ressources en numérotation téléphonique ») et `5750609f88ee3808bf84e966` (« Identifiants de communications électroniques ») le 29/10/2026.
- **Accès** : `https://extranet.arcep.fr/uploads/MAJNUM.csv` (environ 1,1 Mo) et `https://extranet.arcep.fr/uploads/identifiants_CE.csv` (environ 170 Ko), URL fixes, exports complets (pas de delta).
- **Format** : « ANSI » d'après l'éditeur, lu en Windows-1252 (aucun octet de 0x80 à 0x9F observé : identique à Latin-1 sur les fichiers actuels), séparateur `;`, fin de ligne `CRLF`, champs entre guillemets seulement quand nécessaire, dates `JJ/MM/AAAA`.
  - MAJNUM : `EZABPQM;Tranche_Debut;Tranche_Fin;Mnémo;Territoire;Date_Attribution`. Bornes de même longueur et commençant par le préfixe : 10 chiffres (20 631 tranches), 4 ou 6 chiffres pour les numéros courts (`3620`, `118218`), 13 ou 14 chiffres pour les numéros `070` des objets connectés. Aucune tranche ne chevauche une autre. `Territoire` : `Métropole`, `National`, `Guadeloupe, St Martin et St Barthélemy`, `La Réunion`, `La Réunion - Océan indien`, `Martinique`, `Guyane`, `Mayotte`, `Saint Pierre et Miquelon`.
  - Opérateurs : `IDENTITE_OPERATEUR;CODE_OPERATEUR;SIRET_ACTEUR;RCS_ACTEUR;ADRESSE_COMPLETE_ACTEUR;ATTRIB_RESS_NUM;DATE_DECLARATION_OPERATEUR`. `CODE_OPERATEUR` est le `Mnémo` de MAJNUM, unique. `SIRET_ACTEUR` contient un SIRET (parfois en groupes), parfois un SIREN seul, vide pour un opérateur étranger.
- **Fraîcheur** : MAJNUM mis à jour après chaque collège de l'ARCEP (fréquence data.gouv.fr : hebdomadaire), dernière attribution observée le 16/10/2026, fichier du 19/10/2026 ; opérateurs hebdomadaire.
- **Référence** (29/10/2026) : 20 931 tranches de 424 opérateurs, tous présents dans le fichier des opérateurs (1 598 opérateurs) ; 20 105 tranches avec un SIREN, 826 d'opérateurs sans SIRET. Une discussion du jeu (12/04/2025) signale dans la mise à jour du 11/04/2025 des bornes en notation scientifique et des « 0 » initiaux perdus, corrigés le 26/05/2025 ; une autre (14/03/2025) des mnémoniques absents du fichier des opérateurs. Un fichier dont une borne n'est pas un nombre, ou dont la longueur sort du plan (un numéro commençant par 0 a au moins 10 chiffres, un numéro court au plus 6), est rejeté ; un attributaire absent du fichier des opérateurs est gardé avec son seul mnémonique.
- **Données personnelles** : le fichier des opérateurs vise des personnes morales mais cite au moins un entrepreneur individuel (non attributaire de numéros). Décision du propriétaire (29/10/2026) : seuls le nom et le SIREN des attributaires d'une tranche entrent dans l'instantané ; l'adresse n'est pas gardée.
- **Usage** : opérateur attributaire de la tranche d'un numéro, numéro hors de toute tranche attribuée. Ne dit rien de l'utilisateur d'un numéro, qui a pu être porté chez un autre opérateur, ni de sa réputation.

## Code pays de l'IBAN

- **Origine** : aucune source téléchargée. Le pays est lu dans l'IBAN lui-même, validé par `github.com/jacoelho/banking` (MIT) d'après le registre ISO 13616 de SWIFT ; le nom français du pays vient des tables CLDR générées dans `golang.org/x/text/language/display` (module sous BSD-3-Clause ; tables CLDR 32 d'après `display.CLDRVersion`, x/text v0.43.0).
- **Usage** : signal `info` `iban_country` (`docs/signals.md`). Ne dit rien du titulaire du compte.

## Domaines officiels (liste de référence)

- **Origine** : aucune source téléchargée. Liste tenue dans le dépôt, `internal/adapter/reference/official/domains.csv`, embarquée dans le binaire : 28 domaines officiels souvent imités (15 services publics, 13 banques de détail), relevés le 02/11/2026. Colonnes : `domain` (domaine enregistrable), `alternates` (autres domaines du même organisme), `names` (noms cherchés comme mots dans un hôte), `of` (l'organisme, tel que le message le cite), `proof` (page de l'organisme qui le prouve).
- **Vérification** : pour chaque ligne, `curl -sIL` mène le domaine au site de l'organisme, et la page `proof` (mentions légales, ou pied de page pour `service-public.gouv.fr`) nomme l'éditeur. Un domaine secondaire n'est listé que s'il redirige vers le domaine principal ou si les mentions légales de l'organisme le nomment. Les mentions légales de `caf.fr` et `labanquepostale.fr`, qui refusent `curl` (protection anti-robots), ont été lues avec un outil de récupération de pages web. `bnpparibas.fr` et `bnpparibas.com`, qui refusent `curl` (403), ont été vérifiés dans un navigateur le 02/11/2026 : tous deux redirigent vers `group.bnpparibas`, dont les mentions légales nomment l'éditeur BNP Paribas S.A. (RCS Paris 662 042 449). Le domaine enregistrable `group.bnpparibas` n'est pas listé : son label générique (« group ») deviendrait un label imité. Le sigle `bnp` est un nom cherché, sur décision du propriétaire (02/11/2026).
- **Écartés** : `cic.com` et `msa.com` appartiennent à d'autres entreprises ; `qonto.com` (éditeur non nommé hors PDF) ; `mabanque.bnpparibas`, dont le label est générique (« ma banque ») ; noms descriptifs (« assurance maladie », « allocations familiales ») et sigles courants (`ants`, `amf`, `msa`, `cic`), qui désignent aussi des sites légitimes.
- **Licence** : faits relevés par nous (noms de domaine et éditeurs), sans reprise de contenu ; aucune licence tierce.
- **Mise à jour** : à la main, ligne vérifiée de la même façon, avec la date dans la PR. Une ligne malformée fait échouer `official.Load`, donc le démarrage de `cmd/api`, `cmd/mcp` et `cmd/eval`.
- **Usage** : signal `attention` `domain_lookalike` (`docs/signals.md`). Ne dit rien de l'exploitant du domaine signalé.

## Plateformes d'hébergement partagées (liste de référence)

- **Origine** : aucune source téléchargée. Liste tenue dans le dépôt, `internal/adapter/reference/hosting/platforms.csv`, embarquée dans le binaire : 16 plateformes qui hébergent les pages ou les liens de leurs utilisateurs sous un chemin de leur propre domaine (réseaux sociaux, messageries, pages de liens, blogs, hébergement de code, raccourcisseurs d'URL), relevées le 02/11/2026. Colonnes : `domain` (domaine enregistrable), `proof` (page ou lien court d'un utilisateur sur la plateforme, en HTTPS).
- **Vérification** : pour chaque ligne, `curl -sL` avec un agent de navigateur sur la page `proof` répond 200 (`twitter.com/X` redirige vers `x.com/X`, `discord.gg/discord-townhall` vers `discord.com/invite/discord-townhall`). `medium.com` et `beacons.ai` refusent `curl` (403) : leur page `proof` a été ouverte dans un navigateur (page de profil affichée). Pour un raccourcisseur, `proof` est un lien court publié par un tiers et `curl` répond 301 vers sa cible : `bit.ly/33fcgUX`, imprimé dans un communiqué de la préfecture de la région Île-de-France, redirige vers `prefectures-regions.gouv.fr` ; `tinyurl.com/ycy49ksl`, lié depuis les pages de `sosnc.gov`, vers une chaîne `youtube.com`. Le chargement vérifie que `proof` est une page HTTPS du domaine de la ligne.
- **Écartés** : aucun. Une plateforme en sous-domaine (`sites.google.com`, `*.notion.site`) n'a pas besoin d'y figurer : seul un hôte égal à son domaine enregistrable, avec ou sans `www.`, peut étendre une page à son site.
- **Licence** : faits relevés par nous (noms de domaine), sans reprise de contenu ; aucune licence tierce.
- **Mise à jour** : à la main, ligne vérifiée de la même façon, avec la date dans la PR. Une ligne malformée fait échouer `hosting.Load`, donc le démarrage de `cmd/api`, `cmd/mcp` et `cmd/eval`.
- **Usage** : garde-fou des listes blanches AMF (`docs/signals.md`) : une entrée avec un chemin sur une de ces plateformes ne couvre pas le site. Aucun signal propre.

## Refusées

- **OpenSanctions** : licence non commerciale. Les listes AMF sont reprises directement à la source.
