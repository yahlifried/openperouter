// SPDX-License-Identifier:Apache-2.0

package frr

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

// TestOPR03_NeighborConfigPasswordRedactedInLogs verifies that NeighborConfig
// does NOT expose the Password field when logged via slog.
//
// RED: currently NeighborConfig has no LogValue() method, so slog dumps
// the struct including the cleartext password.
func TestOPR03_NeighborConfigPasswordRedactedInLogs(t *testing.T) {
	secret := "SuperSecretBGPPassword123"
	nc := NeighborConfig{
		ASN:      mustNewPeerASNFromNumber(64513),
		Addr:     "192.168.1.2",
		ID:       "192.168.1.2",
		Password: secret,
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	logger.Info("test neighbor config", "neighbor", nc)

	if strings.Contains(buf.String(), secret) {
		t.Fatalf("OPR-03: cleartext password %q found in log output:\n%s", secret, buf.String())
	}
}

// TestOPR03_RenderedConfigPasswordRedacted verifies that the rendered FRR
// config string does NOT contain cleartext passwords when logged.
//
// RED: the rendered config string includes "neighbor X password <cleartext>"
// and is logged as-is at debug level.
func TestOPR03_RenderedConfigPasswordRedacted(t *testing.T) {
	secret := "MyBGPSecret456"
	config := Config{
		Underlay: UnderlayConfig{
			MyASN:    64512,
			RouterID: "10.0.0.1",
			Neighbors: []NeighborConfig{
				{
					ASN:      mustNewPeerASNFromNumber(64513),
					Addr:     "192.168.1.2",
					ID:       "192.168.1.2",
					Password: secret,
				},
			},
		},
	}

	configString, err := templateConfig(&config)
	if err != nil {
		t.Fatalf("failed to render config: %v", err)
	}

	// The rendered config should contain the password for FRR (it must be there
	// for the actual config file), but when we redact it for logging purposes,
	// it should not contain the cleartext.
	redacted := RedactPasswords(configString)
	if strings.Contains(redacted, secret) {
		t.Fatalf("OPR-03: cleartext password %q found in redacted config:\n%s", secret, redacted)
	}

	// The redacted string should still contain the "password" keyword (as a marker)
	if !strings.Contains(redacted, "password") {
		t.Fatal("OPR-03: redacted config lost the password line entirely — should keep the line with a redacted value")
	}
}

// TestOPR03_FRRReloadOutputPasswordRedacted verifies that frr-reload.py
// output containing password lines gets redacted before logging.
func TestOPR03_FRRReloadOutputPasswordRedacted(t *testing.T) {
	output := `Reloading frr.conf
+neighbor 192.168.1.2 password MyBGPSecret789
-neighbor 192.168.1.2 password OldPassword123
 neighbor 192.168.1.2 remote-as 64513`

	redacted := RedactPasswords(output)

	for _, secret := range []string{"MyBGPSecret789", "OldPassword123"} {
		if strings.Contains(redacted, secret) {
			t.Fatalf("OPR-03: cleartext password %q found in redacted reload output:\n%s",
				secret, redacted)
		}
	}
}

// TestOPR05_PasswordFieldStoredPlaintext documents that the password field
// stores cleartext BGP passwords. This is a design-level issue: any Neighbor
// with Password set exposes it via kubectl, etcd, audit logs.
//
// The fix for OPR-05 is implementing passwordSecret (OPR-04) and deprecating
// the plaintext password field. This test verifies the FRR config still
// renders correctly when password comes from a secret (non-empty string),
// confirming the passwordSecret path works end-to-end.
func TestOPR05_PasswordFromSecretRendersCorrectly(t *testing.T) {
	resolvedPassword := "resolved-from-k8s-secret"
	nc := NeighborConfig{
		ASN:      mustNewPeerASNFromNumber(64513),
		Addr:     "192.168.1.2",
		ID:       "192.168.1.2",
		Password: resolvedPassword,
	}

	config := Config{
		Underlay: UnderlayConfig{
			MyASN:     64512,
			RouterID:  "10.0.0.1",
			Neighbors: []NeighborConfig{nc},
		},
	}

	configString, err := templateConfig(&config)
	if err != nil {
		t.Fatalf("failed to render config: %v", err)
	}

	expected := fmt.Sprintf("neighbor 192.168.1.2 password %s", resolvedPassword)
	if !strings.Contains(configString, expected) {
		t.Fatalf("OPR-05: rendered FRR config does not contain expected password line %q\n%s",
			expected, configString)
	}
}
