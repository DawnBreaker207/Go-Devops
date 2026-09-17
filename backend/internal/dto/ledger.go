package dto

// LedgerSummaryResponse is the single GROUP BY report the ledger table
// exists for (Phần 2.5): every money type in one place instead of a UNION
// ALL across payments/bookings/user_memberships/point_transactions.
type LedgerSummaryResponse struct {
	Type  string `json:"type"`
	Total int64  `json:"total"`
	Count int64  `json:"count"`
}
