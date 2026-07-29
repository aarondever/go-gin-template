package pagination

type Pagination struct {
	Page     int   `json:"page" form:"page"`
	PageSize int   `json:"page_size" form:"page_size"`
	Total    int64 `json:"total" binding:"-"`
}

func (p *Pagination) Offset() int {
	return (p.GetPage() - 1) * p.Limit()
}

func (p *Pagination) Limit() int {
	switch {
	case p.PageSize > 100:
		p.PageSize = 100
	case p.PageSize <= 0:
		p.PageSize = 10
	}
	return p.PageSize
}

func (p *Pagination) GetPage() int {
	if p.Page <= 0 {
		p.Page = 1
	}
	return p.Page
}
