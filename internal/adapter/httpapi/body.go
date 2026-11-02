// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
)

// errTrailingData means a body holds more than one JSON value.
var errTrailingData = errors.New("data after the JSON value")

// decodeBody decodes into v the JSON body of r, of at most limit bytes, or
// returns the problem refusing it: a body other than JSON, too large, or
// not one JSON value, refused with invalid.
func decodeBody(w http.ResponseWriter, r *http.Request, limit int64, v any, invalid problem) (problem, bool) {
	if !isJSON(r.Header.Get("Content-Type")) {
		return unsupportedMediaType, false
	}
	if err := decodeOne(http.MaxBytesReader(w, r.Body, limit), v); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return contentTooLarge, false
		}
		return invalid, false
	}
	return problem{}, true
}

// isJSON reports whether contentType names application/json, whatever its
// case and parameters.
func isJSON(contentType string) bool {
	mt, _, err := mime.ParseMediaType(contentType)
	return err == nil && mt == "application/json"
}

// decodeOne decodes into v the one JSON value r holds.
func decodeOne(r io.Reader, v any) error {
	dec := json.NewDecoder(r)
	if err := dec.Decode(v); err != nil {
		return err
	}
	switch _, err := dec.Token(); {
	case err == io.EOF:
		return nil
	case err != nil:
		return err
	}
	return errTrailingData
}
