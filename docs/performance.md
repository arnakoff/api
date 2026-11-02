# Performance de l'API

Tests de charge de `GET /v1/check` et budget de latence (#61). Les micro-benchmarks du chemin chaud restent ceux de `go test -bench` (voir `CLAUDE.md`) ; ce document mesure le service entier sous charge.

## Banc de charge

`cmd/loadtest` sert l'API comme `cmd/api` : même handler, mêmes listes lues dans les instantanés stockés, mêmes clés API vérifiées dans PostgreSQL, mêmes limites de débit. Seuls les registres diffèrent : RDAP, SIRENE et BODACC sont rejoués depuis l'enregistrement du corpus (`testdata/corpus/replay`), chaque réponse après `-latency`, pour qu'aucune charge ne les atteigne. Les profils `pprof` sont servis sur `-pprof`, une adresse de boucle locale seulement. Le générateur de charge est [vegeta](https://github.com/tsenart/vegeta), épinglé dans `go.mod` (`go tool vegeta`), version consignée dans `docs/standards.md`.

Déroulé, depuis la racine du dépôt ; une fois les modules et l’image PostgreSQL téléchargés, rien ne sort de la machine :

```sh
docker run -d --rm --name arnakoff-pg -e POSTGRES_PASSWORD=test -e POSTGRES_DB=arnakoff_test \
  -p 127.0.0.1:55432:5432 postgres:18.6
export ARNAKOFF_LOG_KEY=$(openssl rand -hex 32) ARNAKOFF_API_KEY_SECRET=$(openssl rand -hex 32)
export ARNAKOFF_DATABASE_URL='postgres://postgres:test@127.0.0.1:55432/arnakoff_test?sslmode=disable'
export ARNAKOFF_TRUSTED_PROXIES=none ARNAKOFF_RATE_LIMIT_KEY=10000 ARNAKOFF_ADDR=127.0.0.1:8100 ARNAKOFF_LOG_LEVEL=warn
mkdir -p var/load && go build -o var/load/ ./cmd/loadtest ./cmd/admin
var/load/admin migrate
for i in $(seq 1 100); do var/load/admin key create -label load$i; done | grep '^ark_' > var/load/keys.txt
var/load/loadtest -targets http://127.0.0.1:8100 -keys var/load/keys.txt > var/load/targets.http
var/load/loadtest -latency 0 &            # ou -latency 200ms
go tool vegeta attack -targets var/load/targets.http -rate 1000/s -duration 30s -timeout 15s | go tool vegeta report
curl -o var/load/cpu.pprof 'http://127.0.0.1:6060/debug/pprof/profile?seconds=20'   # pendant une attaque
go tool pprof -top -cum var/load/loadtest var/load/cpu.pprof
```

Les cibles sont les 478 entrées du corpus (domaines, e-mails, téléphones, SIREN), une `GET /v1/check` chacune, les clés portées à tour de rôle : une clé est limitée à `ARNAKOFF_RATE_LIMIT_KEY` requêtes par minute (10 000 au plus), donc 100 clés pour dépasser 167 requêtes par seconde sans réponse `429`. `var/` n'est pas versionné : les clés et les cibles n'y sont que locales.

## Mesures du 02/11/2026

Machine : Apple M4 (10 cœurs), 16 Go, macOS, Go 1.27.3 ; PostgreSQL 18.6 en conteneur sur la même machine ; vegeta sur la même machine. Instantanés AMF, AMF PSAN, AMF biens divers et ARCEP ; ACPR désactivée (défaut). Attaques de 30 s à débit constant.

Registres rejoués sans latence (coût propre de l'API) :

| Débit demandé | Débit servi | p50 | p90 | p99 | max | Succès |
|---|---|---|---|---|---|---|
| 100/s | 100/s | 0,98 ms | 1,47 ms | 2,06 ms | 22,2 ms | 100 % |
| 1 000/s | 1 000/s | 0,34 ms | 0,49 ms | 0,92 ms | 3,5 ms | 100 % |
| 4 000/s | 4 000/s | 0,25 ms | 0,52 ms | 0,91 ms | 5,3 ms | 100 % |
| 8 000/s | 8 000/s | 0,50 ms | 0,73 ms | 1,62 ms | 6,5 ms | 100 % |
| 12 000/s | 12 000/s | 0,51 ms | 0,73 ms | 3,01 ms | 33,7 ms | 100 % |
| 16 000/s | 2 559/s | 41 ms | 8,7 s | 15,6 s | 18,4 s | 62 % |

Registres rejoués avec 200 ms de latence chacun :

| Débit demandé | Débit servi | p50 | p90 | p99 | max | Succès |
|---|---|---|---|---|---|---|
| 100/s | 99/s | 201,0 ms | 201,3 ms | 201,8 ms | 214,0 ms | 100 % |
| 1 000/s | 993/s | 200,4 ms | 200,6 ms | 201,0 ms | 208,7 ms | 100 % |
| 4 000/s | 3 974/s | 200,3 ms | 200,4 ms | 201,0 ms | 208,9 ms | 100 % |

Lecture :

- Le coût propre d'une vérification reste sous 2 ms au p99 jusqu'à 8 000 requêtes par seconde. Les registres sont interrogés en parallèle : leur latence s'ajoute une fois, pas une par registre.
- À 16 000 requêtes par seconde, le banc décroche : vegeta n'atteint que 5 172 requêtes par seconde. Sur 58 798 échecs côté client, 30 228 viennent de l'épuisement des ports locaux du générateur (`can't assign requested address`), 10 671 du délai de 15 s, 2 352 de connexions réinitialisées ; le service répond aussi 423 fois `503 authentication_unavailable`. Le service ne consomme que 2,3 s de processeur en 20 s de profil : les requêtes attendent une connexion du pool PostgreSQL de la recherche de clé (`context canceled` dans `FindKey`). Ce palier mesure la limite d'un banc sur un seul hôte autant que celle du service.
- Profil CPU à 8 000 requêtes par seconde, sans latence : la recherche de la clé dans PostgreSQL (`Authenticator.Authenticate` → `Keys.FindKey` → `pgxpool`) prend 4,15 s des 4,95 s passées dans le handler, soit 84 % ; la vérification elle-même (`Check.Check`) 0,22 s et l'écriture JSON 0,19 s. Le goulot est l'aller-retour à la base par requête, pas le moteur de signaux. C'est la mesure qu'attendent #198 (cache des clés valides, durée à valider par le propriétaire) et #197 (flot de clés inconnues) ; aucun correctif n'est fait ici.
- Ces chiffres valent pour cette machine, avec la base et le générateur sur le même hôte. L'hébergeur n'est pas choisi (#70) : le banc est à rejouer sur la machine de production avant le lancement.

## Budget de latence

Proposition, **à valider par le propriétaire** ; seuils mesurés côté client sur `GET /v1/check`, à un débit soutenu de 100 requêtes par seconde :

| Mesure | p50 | p99 |
|---|---|---|
| Coût propre de l'API (registres rejoués sans latence) | ≤ 5 ms | ≤ 25 ms |
| De bout en bout, registres réels | latence du registre le plus lent + 5 ms | latence du registre le plus lent + 25 ms, plafonnée à 10 s (`checkTimeout`, au-delà le registre est déclaré indisponible) |

Les seuils du coût propre laissent une marge d'environ dix fois les mesures ci-dessus, pour une machine de production plus modeste et une base distante. La latence réelle de RDAP, SIRENE et BODACC n'est pas mesurée ici (le banc n'atteint aucun registre) : « à vérifier » avec les journaux de production (#72).
