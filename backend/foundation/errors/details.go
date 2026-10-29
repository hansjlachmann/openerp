package errors

// Details is what lies behind an error, for the user to look at (Show Entries on the error
// dialog): sections of rows with column captions, every value a text already formatted for the
// user — e.g. the G/L entries of a posting that does not balance and their totals
type Details struct {
	Title    string           `json:"title"`
	Sections []DetailsSection `json:"sections"`
}

// DetailsSection is a table of the details: columns, rows and the rows to point out (marked)
type DetailsSection struct {
	Caption string          `json:"caption"`
	Columns []DetailsColumn `json:"columns"`
	Rows    [][]string      `json:"rows"`
	Marked  []int           `json:"marked,omitempty"`
}

// DetailsColumn is a column of a details section; numbers are aligned right
type DetailsColumn struct {
	Caption string `json:"caption"`
	Numeric bool   `json:"numeric,omitempty"`
}

// WithDetails attaches the details behind the error
func (e *AppError) WithDetails(details *Details) *AppError {
	e.Details = details
	return e
}
