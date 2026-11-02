# 0004 — Stockage des clés API et des quotas

- **Statut** : accepté
- **Date** : 02/11/2026

## Contexte

L'API publique demande une clé (#57), applique une limite de débit (#58) et, plus tard, des niveaux d'abonnement mis à jour par Stripe (#82). Il faut garder les clés, leur niveau, leur révocation et les compteurs de quota sur une longue période, en survivant aux redémarrages et aux déploiements. L'ADR 0001 a écarté toute base de données au démarrage et renvoyé ce choix à une ADR dédiée.

Critères : une ou plusieurs instances, sauvegarde, coût, simplicité.

## Options comparées

| Option | Plusieurs instances | Sauvegarde | Coût | Simplicité |
|---|---|---|---|---|
| SQLite, pilote `modernc.org/sqlite` (BSD-3-Clause, sans cgo) | Non : un seul processus écrit | Copie du fichier ; demande un volume persistant | Aucun | Élevée : tout tient dans le binaire |
| PostgreSQL, pilote `github.com/jackc/pgx/v5` (MIT) | Oui | Celle de l'offre gérée, plus une copie hors de l'hébergeur (#75) | Base gérée payante, prix à comparer dans #70 | Moyenne : un service de plus, des migrations |
| Fichier (JSON) | Non | Copie du fichier | Aucun | Faible en pratique : écriture atomique, accès concurrents et requêtes à écrire soi-même |

Le pilote `github.com/mattn/go-sqlite3` (MIT) a été écarté, car il demande cgo. Versions et licences relevées le 02/11/2026 sur proxy.golang.org et dans le fichier `LICENSE` de chaque module : `modernc.org/sqlite` v1.61.1, `github.com/jackc/pgx/v5` v5.12.0.

## Décision

- **PostgreSQL**, accédé avec `github.com/jackc/pgx/v5`. Le propriétaire a choisi PostgreSQL plutôt que SQLite, la recommandation initiale, pour pouvoir lancer plusieurs instances dès le départ et confier la sauvegarde à une offre gérée.
- **Port à l'endroit où il sert** : `internal/app` définit les petites interfaces dont les cas d'usage ont besoin (lire une clé par son empreinte, révoquer une clé, incrémenter un compteur de quota). L'adaptateur `internal/adapter/store/postgres` les implémente. Le domaine ne connaît pas la base.
- **Contenu** : l'empreinte HMAC-SHA256 à clé de chaque clé API, jamais la clé en clair (#57) ; son niveau, ses dates de création et de révocation ; les compteurs de quota par clé et par période. Aucune entrée vérifiée (domaine, e-mail, numéro, IBAN, SIREN) n'y est écrite.
- **Compteurs** : incrément atomique en une seule requête (`INSERT … ON CONFLICT … DO UPDATE`), sûr avec plusieurs instances. La limite par minute (#58) reste en mémoire dans chaque instance, sans aller-retour vers la base sur le chemin de `/check`.
- **Schéma** : migrations SQL versionnées, embarquées dans le binaire et appliquées par une commande explicite, jamais en silence au démarrage de l'API. L'outil de migration est choisi et vérifié dans la PR qui l'introduit.
- **Configuration** : l'URL de connexion vient d'une variable d'environnement, jamais d'un fichier versionné, et n'apparaît jamais dans les journaux. Chaque requête a un délai maximal hérité du `context.Context`.
- **Sécurité par défaut** :
  - connexion chiffrée en TLS avec vérification du certificat et de l'hôte ;
  - l'API se connecte avec un rôle qui ne peut que lire et écrire ses tables, et un rôle distinct applique les migrations ;
  - la clé HMAC est un secret de l'application, jamais stocké en base, pour qu'une copie de la base ne permette pas, seule, de vérifier une clé ;
  - base injoignable : les requêtes authentifiées échouent (`503`, détail RFC 9457) au lieu de passer sans contrôle de clé ni de quota.
- **Tests** : les cas d'usage se testent avec une implémentation en mémoire du port. L'adaptateur PostgreSQL se teste contre une vraie base, fournie par un conteneur de service dans la CI ; en local, ces tests sont ignorés si la variable d'environnement de la base de test est absente. `go test ./...` ne touche donc jamais le réseau sans qu'on le demande.

## Conséquences

- L'hébergeur choisi dans #70 (ADR 0005) doit proposer PostgreSQL géré, dans l'Union européenne.
- La version de PostgreSQL et celle du pilote sont vérifiées avec `/check-versions` et notées dans `docs/standards.md` par la PR qui ajoute la dépendance (#57).
- Les durées de conservation des clés révoquées et des compteurs sont fixées par #73 ; la sauvegarde chiffrée hors de l'hébergeur et la restauration par #75.
- Les instantanés des sources restent sur disque et en mémoire (ADR 0001) : ils se reconstruisent et ne vont pas en base.
- Avec plusieurs instances, la limite par minute s'applique à chaque instance séparément. Si cet écart pose problème, une limite partagée fera l'objet d'une nouvelle ADR.
