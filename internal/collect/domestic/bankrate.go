package domestic

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"

	"arbcn/internal/collect"
	"arbcn/internal/fact"
)

// BOC 人民币存款利率表两跳爬取（§5 Domestic 银行利率行；允许失败降级 → 人工录入通道补位）：
//   ① 索引页 fd31 → 最新一期利率表链接（"人民币存款利率表YYYY-MM-DD" 首个链接即最新）
//   ② 利率表页 → 整存整取档（活期/三个月/半年/一年/二年/三年/五年）挂牌利率
// 页面为静态 HTML（UTF-8 或 GBK），解析 = 去标签 + 正则取值；任一跳失败 → Poll 失败。
// Fact.Ts：页面无时间戳，用本地采集时间；表生效日期（YYYY-MM-DD）落 Src 供陈旧性检查。

// VenueBOC 是 BOC 挂牌利率的 Fact.Venue 值。
const VenueBOC = "boc"

// rateTerms 是整存整取档的挂牌口径（按页面出现顺序，首个出现即整存整取档）。
var rateTerms = []string{"活期", "三个月", "半年", "一年", "二年", "三年", "五年"}

var (
	linkRe = regexp.MustCompile(`href="([^"]+)"[^>]*>\s*人民币存款利率表`)
	dateRe = regexp.MustCompile(`人民币存款利率表(\d{4}-\d{2}-\d{2})`)
	tagRe  = regexp.MustCompile(`<[^>]*>`)
	termRe = regexp.MustCompile(`(活期|三个月|半年|一年|二年|三年|五年)\s*([0-9]+\.[0-9]{1,4})`)
	nbSpRe = regexp.MustCompile(`&nbsp;?`)
)

// BankRate 采集 BOC 挂牌存款利率（Kind=deposit_rate）。
type BankRate struct{ cfg Config }

// NewBankRate 构造银行挂牌利率 collector。
func NewBankRate(cfg Config) *BankRate { return &BankRate{cfg: cfg} }

// Kind 实现 collect.Collector。
func (*BankRate) Kind() string { return fact.KindDepositRate }

// Poll 两跳爬取 BOC 利率表；缺项/链路断 → 失败（本源自退避，人工录入通道补位）。
func (c *BankRate) Poll(ctx context.Context) ([]fact.Fact, error) {
	client := c.cfg.client()
	index, err := collect.GetText(ctx, client, c.cfg.BankRateURL, nil, 4<<20)
	if err != nil {
		return nil, err
	}
	index = decodeHTML(index)
	href, err := latestTableLink(index)
	if err != nil {
		return nil, err
	}
	base, err := url.Parse(c.cfg.BankRateURL)
	if err != nil {
		return nil, fmt.Errorf("bank_rate: bad base URL: %w", err)
	}
	tableURL, err := url.Parse(href)
	if err != nil {
		return nil, fmt.Errorf("bank_rate: bad table link %q: %w", href, err)
	}
	page, err := collect.GetText(ctx, client, base.ResolveReference(tableURL).String(), nil, 4<<20)
	if err != nil {
		return nil, err
	}
	page = decodeHTML(page)
	rates, ok := parseRateTable(page)
	if !ok {
		return nil, fmt.Errorf("bank_rate: %s: incomplete rate table", tableURL)
	}
	tableDate := ""
	if m := dateRe.FindStringSubmatch(page); m != nil {
		tableDate = m[1]
	}
	src := "boc/fimarkets/lilv/fd31"
	if tableDate != "" {
		src += " 表" + tableDate
	}
	ts := time.Now()
	var out []fact.Fact
	for _, term := range rateTerms {
		out = append(out, fact.Fact{
			Kind:   fact.KindDepositRate,
			Venue:  VenueBOC,
			Symbol: term,
			Value:  rates[term],
			Unit:   fact.UnitPctAnnualized,
			Ts:     ts,
			Src:    src,
		})
	}
	return out, nil
}

// latestTableLink 从索引页取首个"人民币存款利率表"链接（列表按时间倒序，首个即最新）。
func latestTableLink(index string) (string, error) {
	m := linkRe.FindStringSubmatch(index)
	if m == nil {
		return "", fmt.Errorf("bank_rate: no rate table link in index")
	}
	return m[1], nil
}

// parseRateTable 去标签后按首个出现顺序取 7 档利率（整存整取档在零存整取档之前，
// 首个出现即挂牌口径）；缺任一档 → ok=false（部分表不可信）。
func parseRateTable(page string) (map[string]float64, bool) {
	text := tagRe.ReplaceAllString(nbSpRe.ReplaceAllString(page, " "), " ")
	rates := make(map[string]float64)
	for _, m := range termRe.FindAllStringSubmatch(text, -1) {
		if _, seen := rates[m[1]]; seen {
			continue
		}
		v, err := strconv.ParseFloat(m[2], 64)
		if err != nil {
			return nil, false
		}
		rates[m[1]] = v
	}
	for _, term := range rateTerms {
		if _, ok := rates[term]; !ok {
			return nil, false
		}
	}
	return rates, true
}

// decodeHTML 页文本解码：UTF-8 直取，否则按 GB18030 解（BOC 历史页面用 GB 系编码）。
func decodeHTML(s string) string {
	b := []byte(s)
	if utf8.Valid(b) {
		return s
	}
	if out, err := simplifiedchinese.GB18030.NewDecoder().Bytes(b); err == nil {
		return string(out)
	}
	return strings.ToValidUTF8(s, "?")
}
