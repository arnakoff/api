# Performance de l'API

Tests de charge de `GET /v1/check` et budget de latence (#61). Les micro-benchmarks du chemin chaud restent ceux de `go test -bench` (voir `CLAUDE.md`) ; ce document mesure le service entier sous charge.

## Banc de charge

`cmd/loadtest` sert l'API comme `cmd/api` : même handler, mêmes listes lues dans les instantanés stockés, mêmes clés API vérifiées dans PostgreSQL, mêmes limites de débit. Seuls les registres diffèrent : RDAP, SIRENE et BODACC sont rejoués depuis l'enregistrement du corpus (`testdata/corpus/replay`), chaque réponse après `-latency`, pour qu'aucune charge ne les atteigne. Les profils `pprof` sont servis sur `-pprof`, une adresse de boucle locale seulement. Le générateur de charge est [vegeta](https://github.com/tsenart/vegeta), épinglé dans `go.mod` (`go tool vegeta`), version consignée dans `docs/standards.md`.

Déroulé, depuis la racine du dépôt ; une fois les modules et l’image PostgreSQL téléchargés, rien ne sort de la machine :

```sh
docker run -d --rm --name arnakoff-pg -e POSTGRES_PASSWORD=test -e POSTGRES_DB=arnakoff_test \
  -p 127.0.0.1:55432:5432 postgres:17.11
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

## Flot de clés inconnues (#197)

Une clé bien formée se fabrique sans rien demander : sa somme de contrôle CRC-32 est publique. Avant #197, chaque clé bien formée mais jamais émise coûtait une recherche dans PostgreSQL. Depuis, chaque adresse cliente (préfixe /64 en IPv6) a un budget de clés refusées, `ARNAKOFF_RATE_LIMIT_AUTH` par minute (30 par défaut). Seule une clé refusée le dépense, une fois refusée (#209) : une clé valide ou une base qui ne répond pas n'y touchent pas, et des clés valides peuvent être recherchées en même temps sans limite de nombre. Des refus arrivés en même temps sont tous comptés : le budget peut devenir négatif, et l'adresse attend alors qu'il soit remboursé au rythme du budget. Le rythme moyen reste donc borné, mais une rafale de clés inconnues envoyées en même temps depuis une adresse au budget non épuisé atteint la base d'un seul coup, avant d'être comptée : le pool de connexions la fait attendre, et l'adresse attend ensuite d'avoir remboursé toute la rafale. Une fois le budget épuisé, les requêtes de cette adresse reçoivent `429` sans recherche.

Mesure du 02/11/2026, même machine et même banc que ci-dessus, registres rejoués sans latence. `ARNAKOFF_TRUSTED_PROXIES=127.0.0.1/32` : chaque cible porte un `X-Forwarded-For`, ce qui donne deux adresses clientes depuis un seul hôte. Pendant 30 s, en même temps : 1 000 requêtes par seconde avec les 100 clés valides depuis `198.51.100.1`, et un flot de 20 000 clés bien formées jamais émises depuis `203.0.113.9`. Réponses aux clés valides :

| Flot de clés inconnues | Version | p50 | p99 | max | Succès | Recherches du flot en base |
|---|---|---|---|---|---|---|
| 4 000/s | avant (`main`) | 0,28 ms | 0,75 ms | 20,7 ms | 100 % | 120 000 |
| 4 000/s | après | 0,29 ms | 1,57 ms | 29,5 ms | 100 % | 44 |
| 12 000/s | avant (`main`) | 0,48 ms | 12,5 ms | 45,4 ms | 100 % | 360 002 |
| 12 000/s | après | 0,30 ms | 0,94 ms | 32,8 ms | 100 % | 44 |

Lecture :

- Depuis une seule adresse, le flot n'atteint plus la base que 44 fois en 30 s : la rafale de 30, puis le budget rendu au rythme de 30 par minute. Toutes les autres requêtes reçoivent `429` en 54 µs au p50.
- À 4 000 clés inconnues par seconde, la base locale suit sans peine, avant comme après. L'écart de p99 entre les deux lignes vient d'une série à l'autre, pas du budget : le banc tourne sur un poste partagé, et une série précédente, avec la même logique avant simplification, avait donné 0,44 ms.
- Avant #197, la base et le générateur sur le même hôte, les clés valides ne recevaient aucun `503`. Leur p99 était cependant multiplié par 17 à 12 000 clés inconnues par seconde. Qu'une base distante, au pool saturé, finisse par renvoyer `503` une fois sa recherche bornée à 3 s dépassée reste « à vérifier » : ce cas n'est pas mesuré ici.
- Le budget ne protège que contre un flot venu d'un petit nombre d'adresses. Un flot réparti sur beaucoup d'adresses atteint encore la base, à raison de 30 recherches par minute et par adresse. Seul le cache des clés valides de #198 rend les clés valides indépendantes de la base. Si le budget suit déjà autant d'adresses qu'il le peut (`ratelimit.DefaultMaxKeys`, 100 000), une adresse nouvelle n'est pas suivie et sa clé est vérifiée en base : un flot réparti ne peut pas fermer la porte à tous les nouveaux clients.
- Coût sur le chemin chaud (`go test -bench`, Apple M4, deux séries de 6 passes contre `main`) : une allocation de plus par requête (`BenchmarkCheck` 40 → 41), la clé de l'adresse cliente. L'écart de temps reste dans le bruit : +4,0 % sur une série, −6,6 % sur l'autre (`BenchmarkCheck` 3,30 µs). `Limiter.Allow` passe de 53,9 à 23,1 ns : son seau à jetons, écrit pour pouvoir rendre une requête, remplace `golang.org/x/time/rate`.

## Cache des clés valides (#198)

Depuis #198, une clé valide trouvée en base est gardée en mémoire 30 s (`app.KeyCacheTTL`, durée validée par le propriétaire le 02/11/2026). Seules les clés actives sont gardées : une clé inconnue ou révoquée, et une base qui ne répond pas, sont vérifiées en base à chaque requête. Depuis #246, une révocation est oubliée tout de suite par chaque instance : un déclencheur sur `api_keys` (migration `0003`) envoie `NOTIFY api_key_revoked` avec l'empreinte de la clé, que chaque API écoute sur une connexion à part (`Keys.WatchRevocations`). Tant que cette écoute n'est pas établie, ou dès qu'elle est perdue, le cache est vidé et ne garde plus rien : chaque clé est vérifiée en base, aux performances d'avant #198. Les 30 s ne bornent plus que la durée de vie d'une entrée en mémoire.

Mesure du 03/11/2026, même machine et même banc que ci-dessus, registres rejoués sans latence, 100 clés valides depuis une seule adresse, `main` puis cette version, une attaque de 30 s par palier :

| Débit demandé | Version | p50 | p90 | p99 | max | Succès |
|---|---|---|---|---|---|---|
| 8 000/s | avant (`main`) | 0,42 ms | 0,58 ms | 0,98 ms | 5,0 ms | 100,00 % |
| 8 000/s | après | 0,057 ms | 0,066 ms | 0,18 ms | 1,6 ms | 100 % |
| 12 000/s | avant (`main`) | 0,48 ms | 0,63 ms | 1,28 ms | 20,8 ms | 99,92 % |
| 12 000/s | après | 0,058 ms | 0,068 ms | 0,20 ms | 9,3 ms | 100 % |
| 16 000/s | avant (`main`) | 0,60 ms | 1,44 ms | 2,57 ms | 37,3 ms | 98,96 % |
| 16 000/s | après | 0,059 ms | 0,083 ms | 0,26 ms | 7,6 ms | 100 % |

Lecture :

- Le p50 est divisé par 7 à 10, le p99 par 5 à 10. À 16 000 requêtes par seconde, le palier où le banc décrochait le 02/11 (avec un pool PostgreSQL saturé), le service suit désormais sans erreur.
- Profil CPU à 8 000 requêtes par seconde, 20 s : 19,3 s de processeur avant, 7,0 s après. La recherche de clé (`Keys.FindKey`) prenait 4,5 s ; elle tombe sous le seuil d'échantillonnage.
- Les échecs de `main` sont tous des `429` du budget de clés refusées (`RateLimit-Policy: "auth";q=30;w=60`), pas du quota des clés : ce budget est dépensé avant la recherche et rendu après, donc plus de 30 recherches en cours depuis une même adresse font refuser des clés valides. Le cache rend ces recherches assez brèves pour que le banc ne le déclenche plus, mais une rafale de clés absentes du cache y restait exposée : corrigé par #209, le budget n'étant plus dépensé que par les refus.

## Budget de latence

Budget **validé par le propriétaire le 02/11/2026** ; seuils mesurés côté client sur `GET /v1/check`, à un débit soutenu de 100 requêtes par seconde :

| Mesure | p50 | p99 |
|---|---|---|
| Coût propre de l'API (registres rejoués sans latence) | ≤ 5 ms | ≤ 25 ms |
| De bout en bout, registres réels | latence du registre le plus lent + 5 ms | latence du registre le plus lent + 25 ms, plafonnée à 10 s (`checkTimeout`, au-delà le registre est déclaré indisponible) |

Les seuils du coût propre laissent une marge d'environ dix fois les mesures ci-dessus, pour une machine de production plus modeste et une base distante. La latence réelle de RDAP, SIRENE et BODACC n'est pas mesurée ici (le banc n'atteint aucun registre) : « à vérifier » avec les journaux de production (#72).
