// SPDX-License-Identifier: AGPL-3.0-only

// Package objectstore keeps the last valid snapshot of each source in an
// S3 bucket, the production store of ADR 0005: Scaleway Object Storage in
// fr-par, shared by the sync job and the API container, which have no
// volume in common.
//
// Each source is one object whose key is the file name of package store
// (amf.snapshot) and whose content is its file format, envelope and digest
// included: Load checks a downloaded object as store.Disk checks a file.
// A PUT replaces an object at once, so a reader gets the previous snapshot
// or the new one, never part of one.
//
// Load remembers the entity tag of the last object it accepted for each
// source and sends it in If-None-Match: while the object is unchanged, the
// bucket answers 304 and the remembered snapshot is returned without a
// download. The API reloads every minute; the sync job writes a few times a
// day.
//
// Requests go path-style to the endpoint host, signed with AWS Signature
// Version 4, through the given HTTP client: in production the platform
// client, which allows that host only and bounds time and response size.
package objectstore
