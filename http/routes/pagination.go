package routes

type pageref struct {
	Href string `json:"href"`
}

type pagination struct {
	First        *pageref `json:"first"`
	Last         *pageref `json:"last"`
	Next         *pageref `json:"next"`
	Previous     *pageref `json:"previous"`
	TotalPages   int      `json:"total_pages"`
	TotalResults int      `json:"total_results"`
}
