// D-040 测试网账户快照数据面（docs/design/04-m3-spec §9.4 S3 扩展）：sim_testnet_accounts
// 读写（upsert + list）。探针成功路径写（source 主键幂等），SimExec 测试网账户区读。
// 纯模拟盘数据（SIMULATED），无任何真实账户/交易路径（D-010 无密钥铁律）。
package pgstore

import (
	"context"
	"encoding/json"
	"errors"

	"arbcn/internal/store"
)

// UpsertTestnetAccount 幂等 upsert（source 主键）；updated_at = DB now()（不信任客户端时钟）。
// Source 必填；Details 空 → 存 '[]'（NOT NULL DEFAULT）。
func (s *Store) UpsertTestnetAccount(ctx context.Context, a store.TestnetAccount) error {
	if a.Source == "" {
		return errors.New("pgstore: testnet account: source required")
	}
	details, err := json.Marshal(a.Details)
	if err != nil {
		return err
	}
	if a.Details == nil {
		details = []byte("[]")
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO sim_testnet_accounts (source, account_alias, equity_usd, details, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (source) DO UPDATE SET
			account_alias = EXCLUDED.account_alias,
			equity_usd    = EXCLUDED.equity_usd,
			details       = EXCLUDED.details,
			updated_at    = now()`,
		a.Source, a.AccountAlias, a.EquityUSD, details)
	return err
}

// ListTestnetAccounts 按 source ASC 返回全部账户快照；无数据 = 空切片。
func (s *Store) ListTestnetAccounts(ctx context.Context) ([]store.TestnetAccount, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT source, account_alias, equity_usd, details, updated_at
		FROM sim_testnet_accounts
		ORDER BY source ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []store.TestnetAccount{}
	for rows.Next() {
		var a store.TestnetAccount
		var raw []byte
		if err := rows.Scan(&a.Source, &a.AccountAlias, &a.EquityUSD, &raw, &a.UpdatedAt); err != nil {
			return nil, err
		}
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &a.Details); err != nil {
				return nil, err
			}
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
