// Package rule：规则引擎（docs/design/02-monitor-architecture.md §4/§7）。
// 声明式规则（PG rules 表 = 配置，非代码）+ 告警状态机
// （armed→active→resolved，状态转变才写 alerts 行）。
//
// 条件表达式文法（§4 Cond；解析校验：坏表达式在 New 时即报错）：
//
//	cond    := or ("||" or)*
//	or      := cmp ("&&" cmp)*
//	cmp     := operand ("<" | "<=" | ">" | ">=") operand
//	operand := number | number "*" agg | agg
//	agg     := name "_" dur [ "(" venue "," symbol ")" ] [ "@" dur ]
//	name    := avg | last | p25 | chg      dur := [0-9]+ [mhd]
//
// 聚合语义（窗口 [now−W−O, now−O)）：
//   - avg_Nu  窗口均值；last_Nu 窗口最新值；p25_Nu 窗口 25 分位（最近秩）；
//   - chg_Nu  环比 = 最新值 − 紧邻更早的一采集值（W 只约束最新值新鲜度，
//     更早值回溯上限 24h——覆盖 v1 全部轮询间隔）；
//   - @O      窗口前移（跨窗口比较：TRX 穿越"此前 48h 均值" = avg_48h@24h）。
//
// 规则类型两种：①阈值（窗口聚合 vs 常数）；②穿越（跨窗口聚合比较，@O 表达）。
// 两者统一走本求值器，无需 schema 分型。
//
// 显式 (venue,symbol) = 跨实体比较（阶梯陷阱）：含显式 scope 的条件在规则
// 全部事实集上全局评估一次；同一条件内显式/隐式 scope 混用 = 解析错误。
// 任一操作数无数据（undefined）→ 该比较为假（无数据不告警）。
package rule

import (
	"math"
	"sort"
	"time"

	"arbcn/internal/fact"
)

// cmpOp 是比较操作符。
type cmpOp int

const (
	opGT cmpOp = iota
	opLT
	opGE
	opLE
)

func (o cmpOp) apply(a, b float64) bool {
	switch o {
	case opGT:
		return a > b
	case opLT:
		return a < b
	case opGE:
		return a >= b
	case opLE:
		return a <= b
	}
	return false
}

// aggKind 是聚合函数。
type aggKind int

const (
	aggAvg aggKind = iota
	aggLast
	aggP25
	aggChg
)

// aggregate 是一个窗口聚合操作数。
type aggregate struct {
	kind   aggKind
	window time.Duration
	offset time.Duration
	venue  string // 显式 scope（跨实体比较）；空 = 隐式（逐实体）
	symbol string
}

func (a *aggregate) scoped() bool { return a.venue != "" || a.symbol != "" }

// operand 是比较的一侧：常数或缩放聚合。
type operand struct {
	val  float64    // agg == nil 时生效
	agg  *aggregate // nil = 常数
	mult float64    // 聚合系数（默认 1）
}

type cmpExpr struct {
	lhs, rhs operand
	op       cmpOp
}

// condExpr 是解析后的条件（OR of ANDs）。
type condExpr struct {
	ands   [][]cmpExpr
	scoped bool // 含显式 scope 聚合 → 全局模式
}

// lookback 返回本条件的最大查询回溯（chg 的更早值宽限 24h）。
func (c *condExpr) lookback() time.Duration {
	var lb time.Duration
	for _, ands := range c.ands {
		for _, cm := range ands {
			for _, op := range []operand{cm.lhs, cm.rhs} {
				if op.agg == nil {
					continue
				}
				w := op.agg.window + op.agg.offset
				if op.agg.kind == aggChg {
					w += 24 * time.Hour
				}
				if w > lb {
					lb = w
				}
			}
		}
	}
	return lb
}

// eval 对事实集求值；返回 (条件满足, 代表值 = 首个 AND 组首个比较的 lhs)。
func (c *condExpr) eval(fs []fact.Fact, now time.Time) (bool, float64) {
	for _, ands := range c.ands {
		ok := true
		for _, cm := range ands {
			l, okL := operandValue(cm.lhs, fs, now)
			r, okR := operandValue(cm.rhs, fs, now)
			if !okL || !okR || !cm.op.apply(l, r) {
				ok = false
				break
			}
		}
		if ok {
			v, _ := operandValue(ands[0].lhs, fs, now)
			return true, v
		}
	}
	return false, 0
}

func operandValue(op operand, fs []fact.Fact, now time.Time) (float64, bool) {
	if op.agg == nil {
		return op.val, true
	}
	v, ok := aggValue(*op.agg, fs, now)
	if !ok {
		return 0, false
	}
	return op.mult * v, true
}

// aggValue 计算聚合值；无数据返回 ok=false。
func aggValue(a aggregate, fs []fact.Fact, now time.Time) (float64, bool) {
	src := fs
	if a.scoped() {
		src = filterEntity(fs, a.venue, a.symbol)
	}
	win := windowFacts(src, now, a.window, a.offset)
	switch a.kind {
	case aggAvg:
		if len(win) == 0 {
			return 0, false
		}
		var sum float64
		for _, f := range win {
			sum += f.Value
		}
		return sum / float64(len(win)), true
	case aggLast:
		if len(win) == 0 {
			return 0, false
		}
		return win[len(win)-1].Value, true
	case aggP25:
		if len(win) == 0 {
			return 0, false
		}
		vals := make([]float64, len(win))
		for i, f := range win {
			vals[i] = f.Value
		}
		sort.Float64s(vals)
		idx := int(math.Ceil(0.25*float64(len(vals)))) - 1 // 最近秩
		return vals[idx], true
	case aggChg:
		// 环比：最新值 − 紧邻更早的一采集值（更早值任意龄，受查询回溯限制）。
		if len(win) == 0 {
			return 0, false
		}
		last := win[len(win)-1]
		i := sort.Search(len(src), func(i int) bool { return src[i].Ts.After(last.Ts) }) - 1
		if i < 1 {
			return 0, false
		}
		return last.Value - src[i-1].Value, true
	}
	return 0, false
}

// filterEntity 按 (venue,symbol) 精确过滤（显式 scope 聚合用）。
func filterEntity(fs []fact.Fact, venue, symbol string) []fact.Fact {
	out := make([]fact.Fact, 0, len(fs))
	for _, f := range fs {
		if f.Venue == venue && f.Symbol == symbol {
			out = append(out, f)
		}
	}
	return out
}

// windowFacts 截取 [now−W−O, now−O) 的事实（输入按 ts 升序，来自 QueryFacts）。
func windowFacts(fs []fact.Fact, now time.Time, w, o time.Duration) []fact.Fact {
	from := now.Add(-w - o)
	to := now.Add(-o)
	lo := sort.Search(len(fs), func(i int) bool { return !fs[i].Ts.Before(from) })
	hi := sort.Search(len(fs), func(i int) bool { return !fs[i].Ts.Before(to) })
	return fs[lo:hi]
}
