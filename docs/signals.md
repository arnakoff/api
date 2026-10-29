# Signaux

Liste publique des signaux. Un code est stable : le renommer passe par `api-change`.

| Code | Entrées | Niveau | Source | Condition |
|---|---|---|---|---|
| `amf_blacklist` | domaine, e-mail | alerte | Liste noire AMF | Entrée présente |
| `acpr_blacklist` | domaine, e-mail | alerte | Liste noire ACPR | Entrée présente |
| `domain_recent` | url, domaine, e-mail | attention | RDAP | Domaine créé il y a moins de 30 jours |
| `company_closed` | siren | attention | SIRENE | Établissement siège fermé |
| `company_procedure` | siren | attention | BODACC | Procédure collective publiée |

Signaux prévus, non implémentés : imitation d'un domaine officiel ou d'une marque, IBAN étranger pour une société française, numéro non attribué selon l'ARCEP.
