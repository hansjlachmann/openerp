package types

// APIResponse is the standard API response wrapper
type APIResponse struct {
	Success  bool                   `json:"success"`
	Data     interface{}            `json:"data,omitempty"`
	Error    string                 `json:"error,omitempty"`
	Captions *CaptionData           `json:"captions,omitempty"`
	Meta     map[string]interface{} `json:"meta,omitempty"`
}

// CaptionData contains translated captions for tables and fields
type CaptionData struct {
	Table   string                       `json:"table,omitempty"`
	Fields  map[string]string            `json:"fields,omitempty"`
	Options map[string]map[string]string `json:"options,omitempty"`
}

// ListRequest represents a list query with filters and pagination
type ListRequest struct {
	Filters   []Filter `json:"filters"`
	SortBy    string   `json:"sort_by"`
	SortOrder string   `json:"sort_order"` // "asc" or "desc"
	Page      int      `json:"page"`
	PageSize  int      `json:"page_size"`
}

// Filter represents a single filter condition
type Filter struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // eq, ne, gt, gte, lt, lte, like, in, between
	Value    interface{} `json:"value"`
	Value2   interface{} `json:"value2,omitempty"` // For BETWEEN operator
}

// ListResponse represents a paginated list response
type ListResponse struct {
	Records  interface{} `json:"records"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// ValidationRequest represents a field validation request
type ValidationRequest struct {
	Field string      `json:"field"`
	Value interface{} `json:"value"`
}

// SessionResponse represents the current session information
type SessionResponse struct {
	Database     string `json:"database"`
	Company      string `json:"company"`
	UserID       string `json:"user_id"`
	UserName     string `json:"user_name"`
	UserFullName string `json:"user_full_name"`
	Language     string `json:"language"`
}

// CodeunitRequest represents a codeunit execution request
type CodeunitRequest struct {
	Params map[string]interface{} `json:"params"`
}

// NewSuccessResponse creates a successful API response
func NewSuccessResponse(data interface{}) *APIResponse {
	return &APIResponse{
		Success: true,
		Data:    data,
	}
}

// NewSuccessResponseWithCaptions creates a successful API response with captions
func NewSuccessResponseWithCaptions(data interface{}, captions *CaptionData) *APIResponse {
	return &APIResponse{
		Success:  true,
		Data:     data,
		Captions: captions,
	}
}

// NewErrorResponse creates an error API response
func NewErrorResponse(errMsg string) *APIResponse {
	return &APIResponse{
		Success: false,
		Error:   errMsg,
	}
}
