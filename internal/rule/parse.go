// 条件表达式解析器（文法见 cond.go 包注释）：递归下降 + 全量校验，
// 坏表达式在 New 时即报错（启动 fail fast）。
package rule

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// parseCond 解析条件串；语法/语义错误返回 error。
func parseCond(s string) (*condExpr, error) {
	p := &parser{toks: tokenize(s)}
	c, err := p.parse()
	if err != nil {
		return nil, fmt.Errorf("rule: cond %q: %w", s, err)
	}
	if tok := p.peek(); tok != "" {
		return nil, fmt.Errorf("rule: cond %q: 位置 %d：多余 token %q", s, p.i+1, tok)
	}
	for _, ands := range c.ands {
		for _, cm := range ands {
			for _, op := range []operand{cm.lhs, cm.rhs} {
				if op.agg != nil && op.agg.scoped() {
					c.scoped = true
				}
			}
		}
	}
	if c.scoped {
		for _, ands := range c.ands {
			for _, cm := range ands {
				for _, op := range []operand{cm.lhs, cm.rhs} {
					if op.agg != nil && !op.agg.scoped() {
						return nil, fmt.Errorf("rule: cond %q: 显式 scope 聚合与隐式聚合混用", s)
					}
				}
			}
		}
	}
	return c, nil
}

// tokenize 切词：括号/逗号按符号切，其余以空白分隔（条件串为简单表达式，无引号）。
func tokenize(s string) []string {
	r := strings.NewReplacer("(", " ( ", ")", " ) ", ",", " , ")
	return strings.Fields(r.Replace(s))
}

// parser 是条件串的递归下降解析器。
type parser struct {
	toks []string
	i    int
}

func (p *parser) next() string {
	if p.i < len(p.toks) {
		t := p.toks[p.i]
		p.i++
		return t
	}
	return ""
}

func (p *parser) peek() string {
	if p.i < len(p.toks) {
		return p.toks[p.i]
	}
	return ""
}

// parse：or := cmp ("&&" cmp)* { "||" or }
func (p *parser) parse() (*condExpr, error) {
	c := &condExpr{}
	for {
		ands, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		c.ands = append(c.ands, ands)
		if p.peek() != "||" {
			return c, nil
		}
		p.next()
	}
}

func (p *parser) parseAnd() ([]cmpExpr, error) {
	var ands []cmpExpr
	for {
		cm, err := p.parseCmp()
		if err != nil {
			return nil, err
		}
		ands = append(ands, cm)
		if p.peek() != "&&" {
			return ands, nil
		}
		p.next()
	}
}

func (p *parser) parseCmp() (cmpExpr, error) {
	lhs, err := p.parseOperand()
	if err != nil {
		return cmpExpr{}, err
	}
	var op cmpOp
	switch p.next() {
	case ">":
		op = opGT
	case "<":
		op = opLT
	case ">=":
		op = opGE
	case "<=":
		op = opLE
	default:
		return cmpExpr{}, fmt.Errorf("位置 %d：期望比较符（> < >= <=）", p.i)
	}
	rhs, err := p.parseOperand()
	if err != nil {
		return cmpExpr{}, err
	}
	return cmpExpr{lhs: lhs, op: op, rhs: rhs}, nil
}

func (p *parser) parseOperand() (operand, error) {
	tok := p.next()
	if tok == "" {
		return operand{}, fmt.Errorf("位置 %d：缺操作数", p.i)
	}
	if v, err := strconv.ParseFloat(tok, 64); err == nil {
		if p.peek() == "*" { // 缩放聚合：number "*" agg
			p.next()
			ag, err := p.parseAggFrom(p.next())
			if err != nil {
				return operand{}, err
			}
			return operand{agg: ag, mult: v}, nil
		}
		return operand{val: v}, nil
	}
	ag, err := p.parseAggFrom(tok)
	if err != nil {
		return operand{}, err
	}
	return operand{agg: ag, mult: 1}, nil
}

// parseAggFrom 解析聚合 token：name_dur[@dur]，后随可选 "(venue,symbol)"。
func (p *parser) parseAggFrom(tok string) (*aggregate, error) {
	base, offTok, hasOff := strings.Cut(tok, "@")
	name, durTok, ok := strings.Cut(base, "_")
	if !ok {
		return nil, fmt.Errorf("位置 %d：%q 不是聚合（期望 name_dur）", p.i, tok)
	}
	var kind aggKind
	switch name {
	case "avg":
		kind = aggAvg
	case "last":
		kind = aggLast
	case "p25":
		kind = aggP25
	case "chg":
		kind = aggChg
	default:
		return nil, fmt.Errorf("位置 %d：未知聚合 %q（支持 avg/last/p25/chg）", p.i, name)
	}
	win, err := parseDur(durTok)
	if err != nil {
		return nil, fmt.Errorf("位置 %d：%w", p.i, err)
	}
	a := &aggregate{kind: kind, window: win}
	if hasOff {
		off, err := parseDur(offTok)
		if err != nil {
			return nil, fmt.Errorf("位置 %d：@后：%w", p.i, err)
		}
		a.offset = off
	}
	if p.peek() == "(" { // 显式 scope："(venue,symbol)"
		p.next()
		a.venue = p.next()
		if p.next() != "," || a.venue == "" {
			return nil, fmt.Errorf("位置 %d：scope 期望 (venue,symbol)", p.i)
		}
		a.symbol = p.next()
		if a.symbol == "" || p.next() != ")" {
			return nil, fmt.Errorf("位置 %d：scope 期望 (venue,symbol)", p.i)
		}
	}
	return a, nil
}

// parseDur 解析 "30d"/"24h"/"15m"；必须为正。
func parseDur(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("非法时长 %q", s)
	}
	n, err := strconv.Atoi(s[:len(s)-1])
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("非法时长 %q", s)
	}
	var unit time.Duration
	switch s[len(s)-1] {
	case 'm':
		unit = time.Minute
	case 'h':
		unit = time.Hour
	case 'd':
		unit = 24 * time.Hour
	default:
		return 0, fmt.Errorf("非法时长单位 %q（支持 m/h/d）", s[len(s)-1:])
	}
	return time.Duration(n) * unit, nil
}
