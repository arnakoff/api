// SPDX-License-Identifier: AGPL-3.0-only

// Command admin administers the API key store (ADR 0004).
//
//	admin migrate [-listen <address>]
//	admin backup [-listen <address>]
//	admin restore -file <dump>
//	admin key create -label <holder> [-env live|test]
//	admin key list
//	admin key revoke -id <id>
//
// migrate applies the embedded SQL migrations not yet applied, under an
// advisory lock: run it with a role that may change the schema, distinct
// from the role of the API, before deploying a version that needs them.
// The API never migrates on start. With -listen, migrate serves /healthz
// and /readyz on that address until it is stopped, /readyz answering 200
// once the migrations are applied and 503 before or after a failure: the
// form a Serverless Container inside the private network runs (#259).
//
// backup copies the API keys, read in one read-only transaction, sealed
// with age for ARNAKOFF_BACKUP_RECIPIENT, to ARNAKOFF_BACKUP_BUCKET on
// Amazon S3 in eu-west-3 (#75): run it with a role that may only read the
// tables. The login requests are left out. With -listen, it serves
// /healthz, /readyz and POST /backup, which runs one backup: the form a
// private Serverless Container runs, woken by a cron trigger.
//
// restore loads a dump, decrypted with age -d, into an empty store: it
// migrates the schema up to the dump's version, loads the rows in one
// transaction, then applies the later migrations, and refuses a database
// that holds keys. Run it with the role of the migrations.
//
// key create draws a key, "ark_live_…" or "ark_test_…", stores its keyed
// digest and prints the key once on standard output: it is never stored and
// cannot be shown again. key list prints every key, its hint (the last four
// characters), its dates and its label, never the key. key revoke revokes a
// key in the store at once; revoking it again keeps the first date. A
// running API is notified of the revocation and refuses the key at once.
//
// The database is ARNAKOFF_DATABASE_URL, which must use
// sslmode=verify-full beyond the local machine, verified against the PEM
// certificates of ARNAKOFF_DATABASE_CA when it is set; keys are digested with
// ARNAKOFF_API_KEY_SECRET, which must differ from ARNAKOFF_LOG_KEY, and
// issued for ARNAKOFF_API_KEY_ENV ("live" by default) unless -env says
// otherwise. Logs go to standard error.
package main
