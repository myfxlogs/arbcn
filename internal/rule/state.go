// 告警状态机（docs/design/02-monitor-architecture.md §4）：
// armed→active→resolved，只有状态转变才写 alerts 行；持续满足期间只更新
// trigger_states.last_value；解除时补发 resolved。
package rule

import (
	"context"
	"fmt"
	"strings"

	"arbcn/internal/store"
)

// match 是命中实体的描述（告警消息用）。
type match struct {
	venue, symbol string
	value         float64
}

// transition 应用状态机转变并落库。返回本轮是否命中。
func (e *Engine) transition(ctx context.Context, r store.Rule, st store.TriggerState, hit bool, matches []match) (bool, error) {
	now := e.now()
	if hit {
		// [对抗测试锚点] 状态未变不告警（持续满足期间只发一次 active）：
		// 删除本分支 → engine_test.go TestStateMachineLifecycle 告警计数断言必红（§11②）。
		if st.State == store.StateActive {
			return true, e.st.PutTriggerState(ctx, store.TriggerState{
				RuleID: r.ID, State: store.StateActive, Since: st.Since, LastValue: matches[0].value,
			})
		}
		if err := e.st.InsertAlert(ctx, store.Alert{
			RuleID: r.ID, Ts: now, Level: r.Level, Message: activeMsg(r.Name, matches),
		}); err != nil {
			return true, fmt.Errorf("rule %q: insert alert: %w", r.Name, err)
		}
		return true, e.st.PutTriggerState(ctx, store.TriggerState{
			RuleID: r.ID, State: store.StateActive, Since: now, LastValue: matches[0].value,
		})
	}
	if st.State != store.StateActive {
		return false, nil // armed/resolved 且未命中 → 无转变，不落库
	}
	if err := e.st.InsertAlert(ctx, store.Alert{
		RuleID: r.ID, Ts: now, Level: r.Level, Message: ruleLabel(r.Name) + " 已解除",
	}); err != nil {
		return false, fmt.Errorf("rule %q: insert resolved alert: %w", r.Name, err)
	}
	return false, e.st.PutTriggerState(ctx, store.TriggerState{
		RuleID: r.ID, State: store.StateResolved, Since: now, LastValue: st.LastValue,
	})
}

// activeMsg 组装激活告警消息（命中实体最多列 3 个，余数省略；中文模板）。
func activeMsg(name string, matches []match) string {
	if len(matches) == 0 {
		return ruleLabel(name) + " 触发"
	}
	n := min(len(matches), 3)
	parts := make([]string, 0, n)
	for _, m := range matches[:n] {
		if m.venue == "" && m.symbol == "" { // 全局模式（跨实体比较）
			parts = append(parts, fmt.Sprintf("%.4g", m.value))
		} else {
			parts = append(parts, fmt.Sprintf("%s@%s=%.4g", m.symbol, m.venue, m.value))
		}
	}
	msg := ruleLabel(name) + " 触发: " + strings.Join(parts, ", ")
	if len(matches) > n {
		msg += ", …"
	}
	return msg
}
