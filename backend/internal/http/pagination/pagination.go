package pagination

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

type Params struct {
	Page     int
	PageSize int
	Sort     string
}

type Result[T any] struct {
	Items    []T   `json:"items"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
	Total    int64 `json:"total"`
}

func Parse(c *gin.Context) Params {
	page := max(1, parseInt(c.Query("page"), 1))
	pageSize := max(1, min(100, parseInt(c.Query("pageSize"), 20)))
	sort := c.Query("sort")
	return Params{Page: page, PageSize: pageSize, Sort: sort}
}

func parseInt(v string, def int) int {
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}
