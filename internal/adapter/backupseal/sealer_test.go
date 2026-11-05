// SPDX-License-Identifier: AGPL-3.0-only

package backupseal_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"filippo.io/age"

	"github.com/arnakoff/api/internal/adapter/backupseal"
	"github.com/arnakoff/api/internal/app"
)

var _ app.BackupSealer = (*backupseal.Sealer)(nil)

func TestSealedBackupOpensWithTheIdentityOnly(t *testing.T) {
	t.Parallel()
	id, err := age.GenerateHybridIdentity()
	if err != nil {
		t.Fatal(err)
	}
	s, err := backupseal.New(id.Recipient().String())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	var sealed bytes.Buffer
	w, err := s.Seal(&sealed)
	if err != nil {
		t.Fatalf("Seal() error = %v", err)
	}
	const plain = "jeanne.martin@example.org"
	if _, err := io.WriteString(w, plain); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sealed.Bytes(), []byte(plain)) {
		t.Fatal("the sealed backup holds the clear text")
	}

	r, err := age.Decrypt(bytes.NewReader(sealed.Bytes()), id)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if got, _ := io.ReadAll(r); string(got) != plain {
		t.Errorf("opened %q, want %q", got, plain)
	}
	other, err := age.GenerateHybridIdentity()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := age.Decrypt(bytes.NewReader(sealed.Bytes()), other); err == nil {
		t.Error("another identity opened the backup")
	}
}

func TestNewRefuses(t *testing.T) {
	t.Parallel()
	id, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	tests := map[string]string{
		"empty":          "",
		"comments only":  "# no key\n",
		"not a key":      "ssh-ed25519 AAAA",
		"a private key":  id.String(),
		"two recipients": id.Recipient().String() + "\n" + id.Recipient().String(),
	}
	for name, recipient := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := backupseal.New(recipient); !errors.Is(err, backupseal.ErrRecipient) {
				t.Errorf("New() error = %v, want ErrRecipient", err)
			}
		})
	}
}
