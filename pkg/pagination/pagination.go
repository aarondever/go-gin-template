package pagination

type Pagination struct {
	Page     int   `json:"page" form:"page"`
	PageSize int   `json:"page_size" form:"page_size"`
	Total    int64 `json:"total" binding:"-"`
}

func (p *Pagination) GetLimit() int {
	if p.PageSize <= 0 {
		p.PageSize = 10
	} else if p.PageSize > 1000 {
		p.PageSize = 1000
	}

	return p.PageSize
}

func (p *Pagination) GetOffset() int {
	if p.Page <= 0 {
		p.Page = 1 // Default to page 1
	}

	return (p.Page - 1) * p.GetLimit()
}
