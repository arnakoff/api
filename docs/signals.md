# Signaux

Liste publique des signaux. Un code est stable : le renommer passe par `api-change`.

| Code | Entrées | Niveau | Source | Condition |
|---|---|---|---|---|
| `amf_blacklist` | url, domaine, e-mail | alerte | Liste noire AMF | Site inscrit (voir ci-dessous) ou adresse e-mail inscrite à l'identique |
| `acpr_blacklist` | url, domaine, e-mail | alerte | Liste noire ACPR | Site inscrit (voir ci-dessous) ou adresse e-mail inscrite à l'identique |
| `amf_registered` | url, domaine, e-mail, téléphone, siren, siret | info | Listes blanches AMF (PSAN, biens divers) | Site, adresse, numéro ou SIREN d'une autorisation en cours (voir ci-dessous) |
| `amf_registration_ended` | url, domaine, e-mail, téléphone, siren, siret | info | Liste blanche AMF PSAN | Toutes les autorisations trouvées sont radiées |
| `domain_recent` | url, domaine, e-mail | attention | RDAP | Domaine créé il y a moins de 30 jours |
| `company_closed` | siren | attention | SIRENE | Établissement siège fermé |
| `company_procedure` | siren | attention | BODACC | Procédure collective publiée |

Signaux prévus, non implémentés : imitation d'un domaine officiel ou d'une marque, IBAN étranger pour une société française, numéro non attribué selon l'ARCEP.

## Listes noires

- **Correspondance** : une entrée de site couvre son hôte et tous ses sous-domaines, jamais le domaine parent (`x.wixsite.com` ne signale pas `wixsite.com`). Une entrée `www.X` couvre le site `X`, sauf si `X` est un suffixe public. Une adresse e-mail est comparée à l'identique, partie locale comprise ; le domaine est normalisé.
- **Pages** : une entrée qui vise une page précise (`calendly.com/amf-france`) ne signale pas son site ; elle n'est pas lue tant que l'entrée vérifiée ne garde pas le chemin de l'URL.
- **Formulation** : « Figure sur la liste noire de l'AMF (catégorie Forex) depuis le 08/10/2026. » La catégorie est celle de l'éditeur, les `_` de l'AMF lus comme des espaces ; « catégories A, B » s'il y en a plusieurs, sans parenthèse s'il n'y en a aucune.
- **Inscriptions multiples** : plusieurs inscriptions dans une même liste donnent un seul signal, avec chaque catégorie une fois et la date la plus ancienne. Une entrée inscrite sur les deux listes donne deux signaux, un par source.
- **Date d'observation** : `observed_at` est l'heure de téléchargement de l'instantané en service.
- **Liens** : AMF `https://www.amf-france.org/fr/espace-epargnants/proteger-son-epargne/listes-noires-et-mises-en-garde`, ACPR `https://www.abe-infoservice.fr/liste-noire` (vérifiés le 30/10/2026).

## Listes blanches

- **Correspondance** : comme pour les listes noires, une entrée de site couvre son hôte et ses sous-domaines, jamais le domaine parent, et une entrée `www.X` couvre `X`. Une entrée de site avec un chemin (`https://www.coinhouse.com/fr/`, 82 entrées PSAN au 30/10/2026) couvre son site : la liste donne le site du prestataire, souvent sa page d'accueil en français (décision du propriétaire, 30/10/2026). Adresse et numéro de téléphone comparés à l'identique une fois normalisés ; un SIREN couvre aussi les SIRET de l'entreprise.
- **Formulation** : PSAN « Figure sur la liste des prestataires sur actifs numériques de l'AMF : COINHOUSE SAS (Agrément MICA) depuis le 07/05/2026. » ; biens divers « Figure sur la liste des offres de biens divers enregistrées par l'AMF : ECOTREE (n° D-18-01) depuis le 26/02/2018. » Le nom est celui de l'éditeur ; chaque couple nom et autorisation une fois, la date la plus ancienne des autorisations en cours. Un nom d'offre long peut dépasser 160 caractères.
- **Conseil** de `amf_registered` : « Des sites imitent des acteurs enregistrés : vérifie l'adresse exacte sur le site de l'AMF avant de verser de l'argent. » L'usurpation d'acteurs enregistrés est une catégorie des listes noires.
- **Radiations** : une autorisation PSAN `Radié` est terminée à sa date de fin. Si une autorisation en cours existe, seules les autorisations en cours sont citées ; sinon `amf_registration_ended` : « A figuré sur la liste des prestataires sur actifs numériques de l'AMF : COINHOUSE SAS (Enregistrement) du 17/03/2020 au 07/05/2026. », du premier début à la dernière fin (décision du propriétaire, 30/10/2026). Niveau `info` : une radiation suit souvent la fin de la période transitoire MiCA, pas une sanction.
- **Jamais une garantie** : figurer sur une liste blanche ne dit rien du site qui imite un acteur listé ; le signal reste `info`.
- **Lien** : `https://www.amf-france.org/fr/espace-epargnants/proteger-son-epargne/listes-blanches` (vérifié le 30/10/2026), pour les deux listes.

## Âge du domaine

- **Condition** : `domain_recent` quand le registre RDAP du domaine enregistrable (celui de l'adresse e-mail pour un e-mail) publie une date d'enregistrement de moins de 30 jours (`signal.RecentDomainAge`) au moment de la vérification. Un domaine de 30 jours pile n'est plus récent. Pas de signal si le registre ne publie pas de date, publie une date postérieure à la vérification, ou ne répond pas.
- **Formulation** : « Le domaine exemple.fr a été créé le 18/10/2026, il y a 12 jours (registre RDAP). » L'âge est en jours entiers : « il y a 1 jour », « il y a moins d'un jour ». La date est donnée dans le fuseau de l'horloge de vérification.
- **Conseil** : « Un site créé il y a quelques jours n'a pas d'historique : vérifie qui l'exploite avant de verser de l'argent. »
- **Lien** : `source_url` est l'adresse RDAP interrogée (`https://rdap.nic.fr/domain/exemple.fr`), réponse JSON publique où la date se vérifie (décision du propriétaire, 30/10/2026).
- **Date d'observation** : `observed_at` est l'heure de la vérification ; la réponse du registre peut venir du cache (24 h au plus), la date d'enregistrement d'un domaine ne changeant pas.
- **Niveau** `attention` : un domaine récent n'est qu'un indice, beaucoup de sites légitimes le sont.
