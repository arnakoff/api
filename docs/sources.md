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

## Refusées

- **OpenSanctions** : licence non commerciale. Les listes AMF sont reprises directement à la source.
