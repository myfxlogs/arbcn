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
	// ref_price 漂移 >2% 独立触发。理由写「生成参考价 → 确认时刻参考价 + 漂移比例 +
	// 阈值」，方向由箭头表达（自明，业主反馈原「ref_price 漂移 x%」看不到前后数值）。
	if drift := math.Abs(curRef-genRef) / genRef; drift > 0.02 {
		return true, fmt.Sprintf("SPREAD_DRIFT: 参考价 %v → %v（漂移 %+.2f%%，超阈值 ±2%%）", genRef, curRef, drift*100)
	}
	// 预期年化变化 >20% 独立触发（分母 genSpread 已保证非零）。理由写「生成预期年化 →
	// 确认时刻年化 + 方向（回落/上行）+ 变化比例 + 阈值」——业主反馈原「预期年化变化
	// 53.28%」被误读为「涨到 53.28%」，实为回落 53%，故显式给两值 + 方向词。
	if spread := math.Abs(curSpread-genSpread) / genSpread; spread > 0.20 {
		dir := "回落"
		if curSpread > genSpread {
			dir = "上行"
		}
		return true, fmt.Sprintf("SPREAD_DRIFT: 预期年化 %.2f%% → %.2f%%（%s %.2f%%，超阈值 ±20%%）", genSpread, curSpread, dir, spread*100)
	}
	return false, ""
}
