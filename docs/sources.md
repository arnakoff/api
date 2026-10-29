# Sources de données

Inventaire contraignant : une source absente de ce fichier n'est pas utilisée. Relevés du 29/10/2026 (étude de faisabilité).

## Liste noire AMF

- **Éditeur** : Autorité des marchés financiers.
- **Licence** : Licence Ouverte 2.0 (`lov2`), réutilisation commerciale autorisée.
- **Accès** : fichier CSV sur data.gouv.fr, jeu `641d5dab1d84b530f7720b08`. URL du fichier horodatée (`export_od_listesnoires_<début>_<fin>.csv`) : la résoudre via `https://www.data.gouv.fr/api/1/datasets/641d5dab1d84b530f7720b08/`.
- **Format** : UTF-8 avec BOM, séparateur `;`, colonnes `nom;categorie;date_inscription` (date ISO). Catégories multiples séparées par `|`.
- **Fraîcheur** : quotidienne annoncée et observée.
- **Référence** : 3 499 lignes (2 550 domaines, 919 e-mails, 30 noms) ; 350 à 430 inscriptions par an.
- **Données personnelles** : non (sites, adresses génériques, sociétés).

## Listes blanches AMF

- **Éditeur** : AMF. **Licence** : Licence Ouverte 2.0.
- **Accès** : data.gouv.fr, jeux `653135cf052c9a87787413c2` (PSAN, prestataires sur actifs numériques) et `65709c9cae162fbc34e7ec05` (offres en biens divers).
- **Fraîcheur** : quotidienne.
- **Usage** : signal `info` « acteur enregistré auprès de l'AMF », jamais une garantie.

## Liste noire ACPR (ABE Info Service)

- **Éditeur** : ACPR et AMF, via Assurance Banque Épargne Info Service.
- **Licence** : **non indiquée, à obtenir avant tout usage commercial** (issue réservée au propriétaire).
- **Accès** : `https://www.abe-infoservice.fr/fr/abeis-liste-noire.csv`.
- **Format** : UTF-8 avec BOM, séparateur `,`, colonnes `"URL / Mails",Dénomination,Catégorie(s),"Date d'inscription"` (date `JJ/MM/AAAA`), catégories multiples séparées par `, `.
- **Fraîcheur** : non annoncée ; dernière inscription observée le 26/10/2026.
- **Référence** : 11 531 lignes ; elle reprend aussi des entrées AMF (1 917 en commun). Union AMF et ACPR : 13 076 entrées uniques.

## SIRENE (API Recherche d'entreprises)

- **Éditeur** : DINUM, données INSEE. **Licence** : Licence Ouverte 2.0.
- **Accès** : `https://recherche-entreprises.api.gouv.fr/search?q=<siren>`, sans clé, quotas publics à respecter.
- **Champs utilisés** : `nom_complet`, `siege.etat_administratif` (`A` active, `F` fermée), `date_creation`.

## BODACC

- **Éditeur** : DILA. **Licence** : Licence Ouverte.
- **Accès** : `https://bodacc-datadila.opendatasoft.com/api/explore/v2.1/catalog/datasets/annonces-commerciales/records`, filtre `registre like "<siren>"`, sans clé.
- **Champs utilisés** : `dateparution`, `familleavis_lib` (procédures collectives, radiations).

## RDAP (âge des domaines)

- **Éditeurs** : AFNIC pour `.fr` (`https://rdap.nic.fr/domain/<d>`), registres via l'amorçage IANA pour les autres extensions (`https://rdap.org/domain/<d>` ou le serveur du registre).
- **Licence** : données d'enregistrement publiques ; seuls les événements (`registration`, `last changed`) sont conservés, jamais les contacts.
- **Champs utilisés** : `events[].eventAction == "registration"`.
- **Limites** : quotas par registre (`429` observé sur `rdap.org`) : mise en cache obligatoire.

## Numérotation ARCEP (MAJNUM)

- **Éditeur** : ARCEP. **Licence** : à confirmer sur le site de l'ARCEP.
- **Accès** : `https://extranet.arcep.fr/uploads/MAJNUM.csv` (environ 1,1 Mo).
- **Format** : Latin-1, séparateur `;`, colonnes `EZABPQM;Tranche_Debut;Tranche_Fin;Mnémo;Territoire;Date_Attribution`.
- **Usage** : opérateur attributaire d'une tranche, numéro non attribué. Ne dit rien de la réputation d'un numéro.

## Refusées

- **OpenSanctions** : licence non commerciale. Les listes AMF sont reprises directement à la source.
