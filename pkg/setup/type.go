package setup

type globalBan struct {
	Code    int64     `json:"code"`
	Status  string    `json:"status"`
	Content []content `json:"content"`
}

type content struct {
	ID        int64       `json:"ID"`
	CreatedAt string      `json:"CreatedAt"`
	UpdatedAt string      `json:"UpdatedAt"`
	DeletedAt interface{} `json:"DeletedAt"`
	Reason    string      `json:"Reason"`
}
