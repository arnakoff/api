# Sources de données

Inventaire contraignant : une source absente de ce fichier n'est pas utilisée. Relevés du 29/10/2026 (étude de faisabilité).

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
- **Usage** : signal `info` « acteur enregistré auprès de l'AMF », jamais une garantie ; un prestataire `Radié` reste listé avec sa date de fin.

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
