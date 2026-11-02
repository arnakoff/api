# Signaux

Liste publique des signaux, servie aussi par `GET /v1/signals`. Un code est stable : le renommer passe par `api-change`. Le tableau est généré depuis le catalogue (`internal/domain/signal/catalog.go`) par `go generate ./...` ; un test échoue s'il n'est pas à jour. Une URL est vérifiée comme son domaine (`domain`) ; un code donné par plusieurs sources a une ligne par source. Les sections suivantes détaillent les conditions.

<!-- Début du tableau généré depuis le catalogue (go generate ./...) : ne pas modifier à la main. -->
| Code | Entrées | Niveau | Source | Condition |
|---|---|---|---|---|
| `amf_blacklist` | domain, email | alerte | AMF | Site inscrit ou adresse e-mail inscrite à l'identique |
| `acpr_blacklist` | domain, email | alerte | ACPR | Site inscrit ou adresse e-mail inscrite à l'identique |
| `amf_registered` | domain, email, phone, siren, siret | info | AMF PSAN | Site, adresse, numéro ou SIREN d'une autorisation en cours |
| `amf_registered` | domain, email, phone, siren, siret | info | AMF biens divers | Site, adresse, numéro ou SIREN d'une autorisation en cours |
| `amf_registration_ended` | domain, email, phone, siren, siret | info | AMF PSAN | Toutes les autorisations trouvées sont radiées |
| `domain_recent` | domain, email | attention | RDAP | Domaine créé il y a moins de 30 jours |
| `domain_lookalike` | domain, email | attention | Domaines officiels | Domaine qui ressemble à un domaine officiel de la liste de référence |
| `phone_operator` | phone | info | ARCEP | Numéro dans une tranche attribuée |
| `phone_unassigned` | phone | attention | ARCEP | Numéro du plan français hors de toute tranche attribuée |
| `company_closed` | siren, siret | attention | SIRENE | Établissement siège fermé |
| `company_procedure` | siren, siret | attention | BODACC | Avis publié de procédure collective, radiation, conciliation ou rétablissement professionnel |
| `iban_country` | iban | info | IBAN | Pays de tenue du compte, d'après le code pays de l'IBAN |
<!-- Fin du tableau généré. -->

Signaux prévus, non implémentés : IBAN étranger pour une société française.

## Listes noires

- **Correspondance** : une entrée de site couvre son hôte et tous ses sous-domaines, jamais le domaine parent (`x.wixsite.com` ne signale pas `wixsite.com`). Une entrée `www.X` couvre le site `X`, sauf si `X` est un suffixe public. Une adresse e-mail est comparée à l'identique, partie locale comprise ; le domaine est normalisé.
- **Pages** : une entrée qui vise une page précise (`calendly.com/amf-france`) ne signale pas son site ; elle n'est pas lue tant que l'entrée vérifiée ne garde pas le chemin de l'URL.
- **Formulation** : « Figure sur la liste noire de l'AMF (catégorie Forex) depuis le 08/10/2026. » La catégorie est celle de l'éditeur, les `_` de l'AMF lus comme des espaces ; « catégories A, B » s'il y en a plusieurs, sans parenthèse s'il n'y en a aucune.
- **Inscriptions multiples** : plusieurs inscriptions dans une même liste donnent un seul signal, avec chaque catégorie une fois et la date la plus ancienne. Une entrée inscrite sur les deux listes donne deux signaux, un par source.
- **Date d'observation** : `observed_at` est l'heure de téléchargement de l'instantané en service.
- **Liens** : AMF `https://www.amf-france.org/fr/espace-epargnants/proteger-son-epargne/listes-noires-et-mises-en-garde`, ACPR `https://www.abe-infoservice.fr/liste-noire` (vérifiés le 30/10/2026).

## Listes blanches

- **Correspondance** : comme pour les listes noires, une entrée de site couvre son hôte et ses sous-domaines, jamais le domaine parent, et une entrée `www.X` couvre `X`. Une entrée de site avec un chemin (`https://www.coinhouse.com/fr/`, 82 entrées PSAN au 30/10/2026) couvre son site : la liste donne le site du prestataire, souvent sa page d'accueil en français (décision du propriétaire, 30/10/2026). Garde-fou (décision du propriétaire, 02/11/2026) : seulement quand l'hôte est le domaine enregistrable, avec ou sans `www.`, et n'est pas une plateforme d'hébergement partagée de la liste de référence (`docs/sources.md`). Sinon l'entrée ne couvre rien : `sites.google.com/view/x` ou `linktr.ee/x` ne couvrent pas toute la plateforme. Au 30/10/2026, aucune des 82 entrées n'est écartée. Adresse et numéro de téléphone comparés à l'identique une fois normalisés ; un SIREN couvre aussi les SIRET de l'entreprise.
- **Formulation** : PSAN « Figure sur la liste des prestataires sur actifs numériques de l'AMF : COINHOUSE SAS (Agrément MICA) depuis le 07/05/2026. » ; biens divers « Figure sur la liste des offres de biens divers enregistrées par l'AMF : ECOTREE (n° D-18-01) depuis le 26/02/2018. » Le nom est celui de l'éditeur ; chaque couple nom et autorisation une fois, la date la plus ancienne des autorisations en cours. Un nom d'offre long peut dépasser 160 caractères.
- **Conseil** de `amf_registered` : « Des sites imitent des acteurs enregistrés : vérifie l'adresse exacte sur le site de l'AMF avant de verser de l'argent. » L'usurpation d'acteurs enregistrés est une catégorie des listes noires.
- **Radiations** : une autorisation PSAN `Radié` est terminée à sa date de fin. Si une autorisation en cours existe, seules les autorisations en cours sont citées ; sinon `amf_registration_ended` : « A figuré sur la liste des prestataires sur actifs numériques de l'AMF : COINHOUSE SAS (Enregistrement) du 17/03/2020 au 07/05/2026. », du premier début à la dernière fin (décision du propriétaire, 30/10/2026). Niveau `info` : une radiation suit souvent la fin de la période transitoire MiCA, pas une sanction.
- **Jamais une garantie** : figurer sur une liste blanche ne dit rien du site qui imite un acteur listé ; le signal reste `info`.
- **Lien** : `https://www.amf-france.org/fr/espace-epargnants/proteger-son-epargne/listes-blanches` (vérifié le 30/10/2026), pour les deux listes.

## IBAN

- **IBAN invalide** : un IBAN dont la longueur, le format du pays ou la clé de contrôle sont faux est refusé, jamais vérifié : la future API répondra `422` (décision du propriétaire, 02/11/2026). L'erreur ne cite jamais l'IBAN.
- **Condition** : `iban_country` pour chaque IBAN valide, français compris (décision du propriétaire, 02/11/2026). Le pays est celui du code pays de l'IBAN (registre ISO 13616) ; l'IBAN ne dit rien du titulaire ni de la banque réelle.
- **Formulation** : « Pays de tenue du compte : Lituanie (code LT de l'IBAN). » Le nom du pays est le nom français CLDR (`golang.org/x/text/language/display`) ; la phrase évite la préposition (« en », « au », « aux »), qui dépend du pays. Pas de conseil.
- **Lien** : `source_url` est la page du registre IBAN de SWIFT, `https://www.swift.com/standards/data-standards/iban-international-bank-account-number` (403 sans navigateur le 02/11/2026, à vérifier dans un navigateur).
- **Date d'observation** : `observed_at` est l'heure de la vérification.
- **Données personnelles** : le signal ne cite que le pays, jamais le reste de l'IBAN. La source `IBAN` n'est pas consultée et n'apparaît pas dans l'état des sources du rapport.

## Âge du domaine

- **Condition** : `domain_recent` quand le registre RDAP du domaine enregistrable (celui de l'adresse e-mail pour un e-mail) publie une date d'enregistrement de moins de 30 jours (`signal.RecentDomainAge`) au moment de la vérification. Un domaine de 30 jours pile n'est plus récent. Pas de signal si le registre ne publie pas de date, publie une date postérieure à la vérification, ou ne répond pas.
- **Formulation** : « Le domaine exemple.fr a été créé le 18/10/2026, il y a 12 jours (registre RDAP). » L'âge est en jours entiers : « il y a 1 jour », « il y a moins d'un jour ». La date est donnée dans le fuseau de l'horloge de vérification.
- **Conseil** : « Un site créé il y a quelques jours n'a pas d'historique : vérifie qui l'exploite avant de verser de l'argent. »
- **Lien** : `source_url` est l'adresse RDAP interrogée (`https://rdap.nic.fr/domain/exemple.fr`), réponse JSON publique où la date se vérifie (décision du propriétaire, 30/10/2026).
- **Date d'observation** : `observed_at` est l'heure de la vérification ; la réponse du registre peut venir du cache (24 h au plus), la date d'enregistrement d'un domaine ne changeant pas.
- **Niveau** `attention` : un domaine récent n'est qu'un indice, beaucoup de sites légitimes le sont.

## Imitation d'un domaine officiel

- **Liste de référence** : services publics et banques de détail souvent imités, chacun vérifié sur une page de l'organisme (`docs/sources.md`, « Domaines officiels »). Un domaine de la liste, ses sous-domaines et les autres domaines vérifiés du même organisme ne sont jamais signalés.
- **Condition** : `domain_lookalike` quand, comparés par leur squelette UTS #39 (Unicode 17.0.0) en minuscules, après décodage des A-labels :
  - le label du domaine enregistrable (`ameli` pour `www.ameli.com`) est celui d'un domaine officiel écrit autrement (homoglyphes : `аmeli` en cyrillique, `arneli`, `ame1i`) ;
  - ou il est à une modification près (insertion, suppression, substitution, inversion de deux lettres voisines) du label d'un domaine officiel d'au moins 6 lettres (`signal.NearLabelLength`) : `lapste`, `lapsote`. Les labels plus courts en sont exclus, trop proches de mots courants (`amelie`) ;
  - ou l'hôte, hors suffixe public, contient un nom listé de l'organisme (colonne `names`) comme suite de mots entiers, séparés par des points et des tirets : `ameli-remboursement.com`, `ameli.fr.securite-compte.com`, `credit-agricole-securite.com`, `ameli.com`. Un nom à l'intérieur d'un mot ne compte pas (`cameliaflowers.fr`). Un label absent des noms (`cic`, `msa`, sigles courants tenus aussi par d'autres entreprises, comme `cic.com`) ne compte que par ses homoglyphes.
  Le premier domaine officiel de la liste qui correspond est cité ; un seul signal.
- **Formulation** : « Le domaine ameli-remboursement.com ressemble à ameli.fr, site officiel de l'Assurance Maladie, mais n'en fait pas partie. » Le domaine cité est l'hôte normalisé (A-labels : un homoglyphe apparaît sous sa forme `xn--`).
- **Conseil** : « Tape toi-même l'adresse officielle plutôt que de suivre un lien reçu. »
- **Lien** : `source_url` est la page de l'organisme, relevée dans la liste, qui montre qu'il exploite le domaine officiel.
- **Date d'observation** : `observed_at` est l'heure de la vérification.
- **Niveau** `attention` : une ressemblance est un indice, pas un fait publié par une autorité. La liste n'est pas une source consultée et n'apparaît pas dans l'état des sources du rapport.

## Entreprises

- **Entrées** : un SIREN, ou un SIRET lu comme le SIREN de son entreprise.
- **Siège fermé** : `company_closed` quand le répertoire SIRENE donne l'établissement siège fermé (`etat_administratif` `F`) : « Le siège de PAYTOP est fermé depuis le 12/03/2024 (répertoire SIRENE). », ou « Le siège de PAYTOP est fermé (répertoire SIRENE). » sans date de fermeture publiée. Le nom est le `nom_complet` de SIRENE.
- **Avis du BODACC** : `company_procedure` quand le BODACC publie, sur l'entreprise, un avis d'une des familles « Procédures collectives », « Radiations », « Procédures de conciliation » ou « Procédures de rétablissement professionnel » (décision du propriétaire, 02/11/2026). Un seul signal, qui cite l'avis le plus récent : « Le BODACC a publié le 05/02/2026 un avis de la famille « Procédures collectives » sur cette entreprise. » Un avis annulé ou rectifié est écarté (voir `docs/sources.md`).
- **Conseil** des deux signaux : « Vérifie que l'entreprise exerce encore avant de lui verser de l'argent. »
- **Liens** : `source_url` de `company_closed` est l'adresse de l'API Recherche d'entreprises interrogée (`https://recherche-entreprises.api.gouv.fr/search?include=siege&minimal=true&per_page=5&q=<siren>`), réponse JSON publique (décision du propriétaire, 02/11/2026) ; celui de `company_procedure` est la page de l'avis sur `www.bodacc.fr` (`url_complete`).
- **Date d'observation** : `observed_at` est l'heure de la vérification ; les réponses peuvent venir du cache (24 h au plus).
- **Niveau** `attention` : le fait est publié par le registre, la conclusion reste à l'utilisateur. Pas de signal si le registre ne répond pas.

## Numérotation téléphonique

- **Couverture** : les numéros du plan de numérotation français, métropole et départements d'outre-mer qui se composent comme des numéros français (`domain.NumberingPlanCovers`). Un numéro étranger laisse la source « non applicable » ; sans instantané ARCEP en service, elle est « indisponible » et ne donne aucun signal.
- **`phone_operator`** (`info`) : « Le numéro appartient à une tranche attribuée à Société française du radiotéléphone (SFR0) le 08/12/2011 (numérotation ARCEP). Le numéro a pu être porté chez un autre opérateur. » Le nom est celui du fichier des opérateurs, suivi du mnémonique ; le mnémonique seul quand le nom manque. La seconde phrase rappelle la portabilité : l'attributaire de la tranche n'est pas forcément l'opérateur du numéro (décision du propriétaire, 02/11/2026).
- **`phone_unassigned`** (`attention`) : « Le numéro n'appartient à aucune tranche attribuée par l'ARCEP : il ne peut pas être en service. »
- **Conseil** des deux signaux : « Un numéro affiché peut être usurpé : rappelle l'organisme au numéro que tu trouves toi-même sur son site officiel. »
- **Lien** : `https://www.data.gouv.fr/datasets/ressources-en-numerotation-telephonique` (vérifié le 02/11/2026), page du jeu qui mène aux fichiers (décision du propriétaire, 02/11/2026).
- **Date d'observation** : `observed_at` est l'heure de téléchargement de l'instantané en service.
- **Niveau** : un numéro hors tranche ne peut pas être en service, mais l'affichage d'un appel peut l'être quand même (usurpation) : `attention`, pas `alerte`.
