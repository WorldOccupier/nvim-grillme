package main

type herdrResponse struct {
	Result herdrResult `json:"result"`
}

type herdrResult struct {
	Pane herdrPane `json:"pane"`
}

type herdrPane struct {
	ID string `json:"pane_id"`
}
