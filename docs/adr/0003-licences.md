# 0003 — Licences

- **Statut** : accepté
- **Date** : 29/10/2026

## Contexte

Arnakoff veut gagner en visibilité et en confiance (un outil qui émet des signaux sur des sites doit pouvoir montrer comment il les calcule), tout en gardant la possibilité de vendre le service hébergé. Les SDK, l'extension et le bot seront intégrés dans le code d'autres projets.

## Décision

- **`arnakoff/api`** : [GNU AGPL-3.0-only](https://www.gnu.org/licenses/agpl-3.0.html). Texte officiel de gnu.org dans `LICENSE`, sans modification. Chaque fichier Go commence par `// SPDX-License-Identifier: AGPL-3.0-only`.
- **Dépôts satellites** (`arnakoff/web`, SDK, `arnakoff/extension`, `arnakoff/bot`) : [MIT](https://opensource.org/license/mit), pour ne freiner aucune intégration.
- **Contributions** : un accord de contribution (CLA) est signé avant la première PR fusionnée d'un contributeur externe. Il permet au propriétaire de proposer plus tard une licence commerciale en parallèle de l'AGPL. Mis en place dans [#11](https://github.com/arnakoff/api/issues/11) : le [texte de l'accord](https://gist.github.com/EnO33/bd7ba34d77c8cd46555bfbdc53a1401d) suit le modèle Contributor Agreements 1.2 de [contributoragreements.org](https://contributoragreements.org/) (licence non exclusive, licence de brevet, sans engagement sur les licences de distribution, droit français), et la signature passe par [cla-assistant.io](https://cla-assistant.io/arnakoff/api), dont le statut `license/cla` est exigé avant toute fusion dans `main`. L'action CLA Assistant Lite, archivée, a été écartée.
- **Dépendances** : uniquement des licences compatibles avec l'AGPL-3.0 (MIT, BSD, ISC, Apache-2.0, MPL-2.0, LGPL, GPL-3.0). Toute autre licence se discute avant l'ajout.
- **Données** : la licence du code ne couvre pas les données. Chaque source garde sa licence (`docs/sources.md`) ; l'attribution est assurée par les champs `source` et `source_url` de chaque signal.

## Pourquoi « -only » et pas « -or-later »

Le propriétaire choisit lui-même s'il adopte une future version de l'AGPL, au lieu de la laisser s'appliquer d'office.

## Conséquences

- Un tiers qui héberge une version modifiée de l'API doit en publier le code source ; les clients qui appellent l'API ne sont pas concernés.
- Certaines entreprises interdisent le code AGPL en interne : elles restent clientes de l'API hébergée.
- Recommandation, pas un avis juridique : une validation par un juriste est prévue avant l'offre commerciale.
