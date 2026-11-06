# 0008 — Offres, prix et quotas

- **Statut** : accepté
- **Date** : 06/11/2026

## Contexte

L'épic #79 ouvre des offres payantes : l'offre gratuite apporte la visibilité, les intégrateurs professionnels (places de marché, e-commerce, fintechs) ont besoin de quotas plus élevés et d'un engagement de service. Le paiement est délégué à Stripe (#81) et le niveau de la clé suit l'abonnement (#82).

L'issue #80 citait « ADR 0006 » ; ce numéro est pris par les durées de conservation, d'où 0008.

Ce que l'API applique aujourd'hui (valeurs par défaut de `internal/platform/config`, modifiables par variable d'environnement) :

| Limite | Valeur par défaut | Portée |
|---|---|---|
| Requêtes par clé API (`ARNAKOFF_RATE_LIMIT_KEY`) | 60 / minute | une seule valeur pour toutes les clés |
| Requêtes sans clé, par adresse (`ARNAKOFF_RATE_LIMIT_IP`) | 30 / minute | routes publiques |
| Démo du site, par adresse / global | 10 / 300 par minute | vérificateur du site |
| Éléments par lot (`ARNAKOFF_BATCH_MAX_ITEMS`) | 50 (plafond 1 000) | un lot de n éléments dépense n |

Limites techniques à connaître avant de choisir :

- la limite par minute vit **en mémoire, par instance** (ADR 0004) : avec plusieurs instances, elle s'applique à chacune séparément ;
- il n'existe **aucun quota mensuel** ni niveau de clé aujourd'hui : un quota par mois demande des compteurs en base (prévus par l'ADR 0004, `INSERT … ON CONFLICT … DO UPDATE`), à construire dans #82 ;
- un engagement de service ne peut pas dépasser celui de l'hébergeur (Scaleway, ADR 0005) ni celui des sources amont (RDAP, SIRENE, BODACC, listes AMF/ACPR) : taux de disponibilité garanti par Scaleway pour Serverless Containers et PostgreSQL géré « à vérifier » sur leurs conditions.

## Options

### Forme des offres

| Option | Pour | Contre |
|---|---|---|
| A. Paliers fixes (gratuit, puis 2 ou 3 paliers mensuels) | simple à facturer dans Stripe, prévisible pour le client | un client entre deux paliers paie trop |
| B. Paiement à l'usage (prix par vérification) | colle à la consommation | facture imprévisible, compteurs et rapport d'usage Stripe à construire |
| C. Paliers avec dépassement facturé | prévisible et sans coupure | le plus complexe à construire et à expliquer |

Retenu : **A** pour la v1 ; B ou C plus tard si la demande le montre.

### Unité du quota

| Option | Remarque |
|---|---|
| Par minute seulement (existant) | aucun travail, mais pas de notion de volume mensuel vendu |
| Par mois + par minute | le mois porte le prix, la minute protège le service ; demande les compteurs en base |

Retenu : par minute seulement.

## Décision

- **Paliers fixes** mensuels (option A).
- **Quota par minute seulement**, par clé, selon le niveau de la clé ; pas de quota mensuel. Le prix achète un débit, pas un volume.
- Quota atteint : refus `429` avec `Retry-After` (comportement actuel), jamais de dépassement facturé.
- Passage à une offre supérieure : immédiat, au prorata calculé par Stripe. Passage à une offre inférieure et résiliation : à la fin du mois payé.
- Impayé : délai de grâce de 14 jours, puis retour au niveau gratuit ; la clé n'est jamais supprimée.
- Une clé sans abonnement actif a le niveau gratuit.

Offres (prix choisis par le propriétaire, sans étude de marché) :

| Offre | Prix HT / mois | Requêtes / minute | Éléments par lot | Engagement de service | Support |
|---|---|---|---|---|---|
| Gratuite | 0 € | 60 | 50 | aucun | issues GitHub |
| Pro | 29 € | 600 | 200 | aucun, au mieux | e-mail, réponse visée sous 2 jours ouvrés |
| Entreprise | 99 € | 3 000 | 1 000 | aucun, au mieux | e-mail, réponse visée sous 1 jour ouvré |

Sans engagement de durée : abonnement mensuel, résiliable à tout moment. Les délais de support sont visés, non garantis ; les périodes de fermeture sont annoncées à l'avance. Aucune offre ne comporte d'engagement de disponibilité chiffré : il viendra dans une nouvelle ADR, une fois la disponibilité mesurée (#72), avec ses exclusions (maintenance, pannes des sources amont, mode dégradé) et ses crédits en cas de manquement.

Le débit vendu est celui de l'API. Il ne garantit pas le débit des registres interrogés à la demande (RDAP, SIRENE, BODACC), qui ont leurs propres limites : SIRENE est limité à 7 requêtes par seconde par instance (`internal/adapter/registry/sirene`), soit 420 par minute, moins qu'une seule clé Pro (#336).

Limites de cette décision :

- la limite par minute s'applique à chaque instance (ADR 0004) : la production en lance jusqu'à 2 (`max_scale = 2`, `deploy/tofu/compute.tf`), une clé peut donc obtenir jusqu'à deux fois son débit, soit 6 000 requêtes par minute pour une clé Entreprise. Une limite partagée entre instances fera l'objet d'une nouvelle ADR si cet écart pose problème ;
- sans quota mensuel, le coût d'une clé au maximum de son débit n'est pas borné par le prix (environ 26 millions de requêtes sur 30 jours pour Pro, 130 millions pour Entreprise) : coût d'hébergement à mesurer avant la mise en vente ;
- une personne peut demander plusieurs clés gratuites avec plusieurs adresses e-mail : à surveiller, un plafond par compte se décidera sur les usages observés ;
- aucune offre payante n'est vendue avant que le cadre juridique et fiscal de la vente soit réglé et les CGV validées par un juriste ; la facturation Stripe (#81) attend ces deux points.

## Questions pour le juriste des CGV

- Structure juridique nécessaire pour vendre, et mention des prix (HT ou TTC) selon le régime de TVA.
- Vente réservée aux professionnels, ou ouverte aux consommateurs avec leur droit de rétractation.
- Limitation de responsabilité : les signaux rapportent des données publiées par des tiers et ne garantissent jamais qu'une entité est sûre.
- Revente ou redistribution des résultats par un client.
- Suspension et révocation d'une clé (abus, impayé).
- Délais de support visés et absence d'engagement de disponibilité.

## Conséquences

- Modifie l'ADR 0004 : elle prévoyait des compteurs de quota par période en base ; aucun n'est nécessaire, la limite par minute en mémoire suffit.
- #82 ajoute le niveau de clé, stocké avec la clé en base (ADR 0004) et mis à jour par les webhooks Stripe, qui en sont la seule source ; la correspondance offre Stripe → niveau ; la limite par minute et la taille de lot de chaque niveau, aujourd'hui une seule valeur pour toutes les clés, deviennent sa configuration, chaque taille de lot restant inférieure ou égale à la limite par minute du niveau. Le prorata exact de Stripe est vérifié dans sa documentation par #82.
- `api/openapi.yaml` décrit les niveaux et les en-têtes `RateLimit` / `RateLimit-Policy` par niveau (`/api-change`).
- `arnakoff/web` publie la page des tarifs et les CGV.
- Le registre des traitements (`arnakoff/conformite`) ajoute la facturation : Stripe comme sous-traitant, données de facturation et leur durée de conservation (obligation comptable, à vérifier).
