package system

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/hansjlachmann/openerp/backend/foundation/codeunits"
)

// Random returns a random whole number from 1 to n (1 when n < 1)
func Random(n int) int {
	if n < 1 {
		return 1
	}
	r, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 1
	}
	return int(r.Int64()) + 1
}

// CreateGuid returns a new random GUID (version 4) in lower case:
// "6f1c…-….-4…-….-…………"
func CreateGuid() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40 // version 4
	b[8] = b[8]&0x3f | 0x80 // variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

var guidPattern = regexp.MustCompile(`^\{?[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\}?$`)

// IsNullGuid reports whether a GUID is blank or all zeros (also when it is not a GUID)
func IsNullGuid(guid string) bool {
	guid = strings.TrimSpace(guid)
	if !guidPattern.MatchString(guid) {
		return true
	}
	return strings.Trim(guid, "{}0-") == ""
}

// UserId returns the user the code runs for (the session's user in a codeunit)
func UserId() string {
	return codeunits.CurrentUserID()
}

// Confirm asks the user a yes/no question in a codeunit (the text filled in with StrSubstNo;
// pass a translated text) and returns the answer; without a dialog (a scheduled run) it is
// false
func Confirm(text string, values ...any) bool {
	return codeunits.Confirm(StrSubstNo(text, values...))
}
