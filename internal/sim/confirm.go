package sim

import (
	"fmt"
	"math"
)

// ConfirmDriftCheck 确认时刻二次门禁（04-m3-spec §10.3 C2 / D-036 G5）。
// 生成 vs 确认时刻的 ref_price 漂移 >2%（或 预期年化变化 >20%）→ 拒单；两条件
// **各自独立触发**（G5"或"关系，note 记具体原因）。返回 (reject, reason)；
// reason 为空 = 通过。
//
// fail-closed（practices #7：确认重查的价可能是 NaN/缺，不得静默放行）：
//   - genRef == 0 → 拒（生成时无参考价 = 数据缺失，spec 显式 fail-closed 项）；
//   - 任一输入非有限（NaN/±Inf）→ 拒；
//   - genSpread == 0 → 拒（年化变化比值分母为零：0/0 = NaN 会静默通过
//     `NaN > 0.20` 恒 false，与 practices #7 同款 NaN 绕过，故显式拒）。
//
// [对抗测试锚点] §10.3：删 `> 0.02` 漂移比较 → TestConfirmDriftRejectsDrift 必红；
// 删 fail-closed 守卫 → TestConfirmDriftFailClosed 必红。
func ConfirmDriftCheck(genRef, genSpread, curRef, curSpread float64) (bool, string) {
	// 有限性守卫：任一输入非有限，或 genRef/genSpread 为零 → fail-closed 拒。
	if genRef == 0 || genSpread == 0 {
		return true, "SPREAD_DRIFT: 生成时参考价/预期年化缺失（fail-closed 拒单）"
	}
	for _, v := range []struct {
		name string
		val  float64
	}{
		{"gen_ref", genRef}, {"gen_spread", genSpread},
		{"cur_ref", curRef}, {"cur_spread", curSpread},
	} {
		if math.IsNaN(v.val) || math.IsInf(v.val, 0) {
			return true, fmt.Sprintf("SPREAD_DRIFT: 确认时刻 %s 非有限值（fail-closed 拒单）", v.name)
		}
	}
	// ref_price 漂移 >2% 独立触发。
	if drift := math.Abs(curRef-genRef) / genRef; drift > 0.02 {
		return true, fmt.Sprintf("SPREAD_DRIFT: ref_price 漂移 %.2f%%", drift*100)
	}
	// 预期年化变化 >20% 独立触发（分母 genSpread 已保证非零）。
	if spread := math.Abs(curSpread-genSpread) / genSpread; spread > 0.20 {
		return true, fmt.Sprintf("SPREAD_DRIFT: 预期年化变化 %.2f%%", spread*100)
	}
	return false, ""
}
