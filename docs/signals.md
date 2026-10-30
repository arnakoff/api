# Signaux

Liste publique des signaux. Un code est stable : le renommer passe par `api-change`.

| Code | Entrées | Niveau | Source | Condition |
|---|---|---|---|---|
| `amf_blacklist` | url, domaine, e-mail | alerte | Liste noire AMF | Site inscrit (voir ci-dessous) ou adresse e-mail inscrite à l'identique |
| `acpr_blacklist` | url, domaine, e-mail | alerte | Liste noire ACPR | Site inscrit (voir ci-dessous) ou adresse e-mail inscrite à l'identique |
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
