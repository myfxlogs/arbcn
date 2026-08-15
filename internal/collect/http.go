package collect

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// CST 是国内行情与日历事件的口径时区（UTC+8，无夏令时；时间戳解析与日期计算共用）。
var CST = time.FixedZone("CST", 8*3600)

// GetJSON GET 并解码 JSON；非 200 或解码失败返回错误；响应体上限 maxBytes 防异常大包。
// header 注入自定义请求头（如新浪 hq API 必须带 Referer）；nil = 无附加头。
func GetJSON(ctx context.Context, client *http.Client, url string, header http.Header, maxBytes int64, out any) error {
	if maxBytes <= 0 {
		maxBytes = 1 << 20
	}
	body, err := getBody(ctx, client, url, header, maxBytes)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("GET %s: decode: %w", url, err)
	}
	return nil
}

// GetText GET 并返回响应文本（非 JSON 端点，如新浪行情脚本）；其余同 GetJSON。
func GetText(ctx context.Context, client *http.Client, url string, header http.Header, maxBytes int64) (string, error) {
	if maxBytes <= 0 {
		maxBytes = 1 << 20
	}
	body, err := getBody(ctx, client, url, header, maxBytes)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// getBody 是 GetJSON/GetText 的公共请求路径（状态码 + 大小上限）。
func getBody(ctx context.Context, client *http.Client, url string, header http.Header, maxBytes int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	for k, vs := range header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes))
	if err != nil {
		return nil, fmt.Errorf("GET %s: read: %w", url, err)
	}
	return body, nil
}
