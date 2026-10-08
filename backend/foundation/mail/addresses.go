package mail

import (
	netmail "net/mail"
	"strings"
)

// AddressListError is the first item of an address list that is not a valid e-mail
// address (Duplicate false) or repeats an earlier address (Duplicate true).
type AddressListError struct {
	Address   string
	Duplicate bool
}

func (e *AddressListError) Error() string {
	if e.Duplicate {
		return "mail: " + e.Address + " is listed more than once"
	}
	return "mail: " + e.Address + " is not a valid e-mail address"
}

// SplitAddressList returns the items of an address list separated by ";" (as BC's
// e-mail fields) or ",", as typed: spaces around items and empty items dropped. It
// does not check them — Send reports a bad item and still sends to the others.
func SplitAddressList(list string) []string {
	var items []string
	for _, item := range strings.FieldsFunc(list, func(r rune) bool { return r == ';' || r == ',' }) {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}
	return items
}

// ParseAddressList checks an address list ("ops@example.com; Hans <hans@example.com>")
// and returns its items as SplitAddressList does. Fails with an *AddressListError on
// an item that is not a valid address or repeats an earlier one (compared without case).
func ParseAddressList(list string) ([]string, error) {
	items := SplitAddressList(list)
	seen := make(map[string]bool, len(items))
	for _, item := range items {
		addr, err := netmail.ParseAddress(item)
		if err != nil {
			return nil, &AddressListError{Address: item}
		}
		key := strings.ToLower(addr.Address)
		if seen[key] {
			return nil, &AddressListError{Address: addr.Address, Duplicate: true}
		}
		seen[key] = true
	}
	return items, nil
}

// JoinAddressList writes address list items in their stored form: "a; b".
func JoinAddressList(items []string) string {
	return strings.Join(items, "; ")
}

// headerAddress writes an address for the To header: "ops@example.com", or the
// RFC 5322 form with the name RFC 2047 encoded ("=?utf-8?q?Bj=C3=B8rn?= <b@example.com>").
func headerAddress(addr *netmail.Address) string {
	if addr.Name == "" {
		return addr.Address
	}
	return addr.String()
}
