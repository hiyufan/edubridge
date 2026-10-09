package jwclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"time"

	"jww/internal/model"
)

const (
	scorePath     = "/studentportal.php/Jxxx/cjxxlb"
	scorePageSize = 9 // 教务系统忽略 rows 参数，每页固定最多 9 条
)

func scoreQuery(page int) url.Values {
	p := strconv.Itoa(page)
	return url.Values{
		"page": {p}, "rows": {strconv.Itoa(scorePageSize)}, "start": {strconv.Itoa((page - 1) * scorePageSize)},
		"p": {p}, "pn": {p}, "_": {strconv.FormatInt(time.Now().UnixMilli(), 10)},
	}
}

func (c *Client) scorePage(ctx context.Context, n int) (*scorePage, error) {
	var (
		p   *page
		err error
	)
	if n == 1 {
		p, err = c.get(ctx, scorePath+"?"+scoreQuery(1).Encode())
	} else {
		p, err = c.postForm(ctx, scorePath, scoreQuery(n))
	}
	if err != nil {
		return nil, err
	}
	var sp scorePage
	if err := json.Unmarshal(p.body, &sp); err != nil {
		if isLoginPage(p.finalURL, string(p.body)) {
			return nil, ErrSessionExpired
		}
		return nil, fmt.Errorf("%w: 第%d页成绩解析失败", ErrUnavailable, n)
	}
	return &sp, nil
}

// Scores 拉取全部成绩（第一页取总数，其余页并发拉取）。
// complete 表示所有分页都成功；同一学年学期同名课程只保留第一条。
func (c *Client) Scores(ctx context.Context) (scores []model.Score, complete bool, err error) {
	first, err := c.scorePage(ctx, 1)
	if err != nil {
		return nil, false, err
	}
	total, _ := strconv.Atoi(first.Total)
	pages := (total + scorePageSize - 1) / scorePageSize
	if pages < 1 {
		pages = 1
	}

	results := make([]*scorePage, pages)
	results[0] = first
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)
	for page := 2; page <= pages; page++ {
		wg.Add(1)
		go func(page int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if sp, err := c.scorePage(ctx, page); err == nil {
				results[page-1] = sp
			}
		}(page)
	}
	wg.Wait()

	complete = true
	seen := make(map[string]bool)
	for _, sp := range results {
		if sp == nil {
			complete = false
			continue
		}
		for _, row := range sp.Rows {
			key := row.Xn + "|" + row.Xq + "|" + row.Kcmc
			if !seen[key] {
				seen[key] = true
				scores = append(scores, row.toScore())
			}
		}
	}
	return scores, complete, nil
}
