// Executor 测试网镜像下单执行接口（D-098 测试网执行层）。
//
// simapi 保持零网络零密钥（D-010，domains_test TestNoRealTradeTokens 把关）：本接口只
// 声明契约，具体执行器由 main 注入（simtestnet.Executor 天然满足，testnet/demo 端点在
// simtestnet 域，本包不出现任何下单端点字面量）。ConfirmSimOrder 在本地成交前对
// testnet/demo venue 逐腿镜像下单（best-effort，execution 成败不影响本地成交，D-037）。
// nil = 镜像关（M3-c 默认，零回归）。
package simapi

import (
	"context"

	"arbcn/internal/simtestnet"
)

// Executor 单方法契约：PlaceOrder 内部完成下单 + 回读成交，ExecResult 即对账数据
// （exchange_order_id / fill_price / fill_qty / status）。
type Executor interface {
	PlaceOrder(ctx context.Context, o simtestnet.ExecOrder) (simtestnet.ExecResult, error)
}
