# Corpus d'évaluation

Mesure la qualité des signaux (`go run ./cmd/eval`, skill `eval-signals`). Deux fichiers CSV en UTF-8, séparateur `,`, en-tête `input,type,label,added_at,note` : l'entrée telle qu'un utilisateur la collerait, son type (`domain`, `email`, `phone`, `iban`, `siren`, `siret`), son étiquette, la date d'inscription (arnaques) ou de vérification (légitimes), et l'origine de la ligne. Aucune ligne n'est écrite de mémoire.

## `scam.csv` : 185 entrées (86 sites, 99 adresses e-mail)

Tirées le 30/10/2026 des instantanés de la liste noire AMF (3 466 lignes) et de la liste noire ACPR (11 455 lignes), synchronisés par `cmd/sync -once` : les 100 inscriptions les plus récentes de chaque liste (sites et adresses), sans doublon. Une entrée présente dans les deux listes (75) cite les deux, avec la date d'inscription la plus ancienne. Les autres viennent seulement de l'ACPR (85) ou seulement de l'AMF (25). La colonne `note` reprend la catégorie de l'éditeur.

La liste ACPR n'a pas encore de licence connue (#30). Le propriétaire a décidé le 30/10/2026 de l'inclure malgré tout dans le corpus.

## `legit.csv` : 293 entrées (241 sites, 6 adresses, 10 numéros, 36 SIREN)

- **AMF PSAN** (245) : chaque site, adresse, numéro et SIREN d'un prestataire au statut `Agréé` dans l'instantané du 30/10/2026. Une faute de l'éditeur (`ttps://www.alphacap.fr/`) est corrigée comme le fait le connecteur.
- **AMF biens divers** (12) : chaque offre enregistrée.
- **Sites publics** (25) : hôtes liés depuis les fiches « vos droits » de `www.service-public.gouv.fr`, relevés le 30/10/2026.
- **Banques** (11) : domaines `.fr` dont le titulaire RDAP AFNIC (`rdap.nic.fr`), relevé le 30/10/2026, est la banque ; il est cité dans `note`.

## `replay/`

Réponses des registres (RDAP, SIRENE, BODACC) enregistrées par une exécution en ligne de `cmd/eval`, rejouées par `-offline` sans réseau (format : `internal/adapter/registry/replay`). Seules les valeurs lues sont gardées (dates, nom de l'entreprise, références des avis), jamais une réponse brute ni un contact. Une nouvelle exécution en ligne remplace chaque réponse obtenue et garde l'ancienne quand le registre échoue ; une requête absente rend sa source indisponible, ce que le rapport compte.

## Agrandir le corpus

Ajouter une ligne quand un cas est manqué ou signalé à tort, avec sa source dans `note`, et le dire dans la PR. Puis relancer `go run ./cmd/eval` en ligne pour enregistrer les réponses des registres.
