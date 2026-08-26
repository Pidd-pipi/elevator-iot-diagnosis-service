package httpapi

import (
	"fmt"
	"net/http"
	"strconv"
)

const (
	// defaultPageLimit 列表接口未显式传 limit 时的默认返回条数。
	defaultPageLimit = 20
	// maxPageLimit 列表接口单页返回条数上限。
	maxPageLimit = 100
)

// pagination 分页参数。
type pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// parsePagination 解析并校验 limit/offset 查询参数。
// limit 缺省取默认值；非法（非整数或负数）返回错误，由调用方映射为 422。
func parsePagination(r *http.Request) (pagination, error) {
	p := pagination{Limit: defaultPageLimit}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			return p, fmt.Errorf("limit 必须为非负整数")
		}
		if n > maxPageLimit {
			n = maxPageLimit
		}
		p.Limit = n
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			return p, fmt.Errorf("offset 必须为非负整数")
		}
		p.Offset = n
	}
	return p, nil
}

// paginate 对已过滤、已排序的切片做分页，返回当前页与总数。
func paginate[T any](items []T, p pagination) ([]T, int) {
	total := len(items)
	if p.Offset >= total {
		return []T{}, total
	}
	end := p.Offset + p.Limit
	if end > total {
		end = total
	}
	return items[p.Offset:end], total
}
