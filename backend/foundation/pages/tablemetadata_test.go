package pages

import "testing"

// Pages name their source table by registry name, the YAML by display name: both must find
// the same metadata.
func TestMetadataKeyMatchesRegistryAndDisplayNames(t *testing.T) {
	for _, pair := range [][2]string{
		{"Customer Ledger Entry", "Customer_ledger_entry"},
		{"Payment Terms", "Payment_terms"},
		{"Job_Queue", "Job_Queue"},
		{"Customer", "Customer"},
	} {
		if metadataKey(pair[0]) != metadataKey(pair[1]) {
			t.Errorf("metadataKey(%q) = %q, metadataKey(%q) = %q", pair[0], metadataKey(pair[0]), pair[1], metadataKey(pair[1]))
		}
	}
}
