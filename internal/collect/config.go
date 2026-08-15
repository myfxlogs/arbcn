package collect

import (
	"fmt"
	"strings"
	"time"
)

// LoadSources 解析 ARBCN_COLLECT_SOURCES（"name=interval" 逗号列表）为启用源清单。
// 未列出的源 = 关闭（启停开关）；interval 覆盖 defaults 的默认间隔。
// spec 为空 → defaults 原样返回（全开）；"off" → 全部关闭。
// 未知源名 / 非法间隔 / 重复列出 → 错误（配置错误 fail fast，不静默降级）。
func LoadSources(spec string, defaults []Named) ([]Named, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return defaults, nil
	}
	if spec == "off" {
		return nil, nil
	}
	known := make(map[string]Named, len(defaults))
	for _, d := range defaults {
		known[d.Name] = d
	}
	var out []Named
	seen := make(map[string]bool, len(defaults))
	for _, part := range strings.Split(spec, ",") {
		name, ds, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			return nil, fmt.Errorf("collect: source %q: want name=interval", part)
		}
		d, ok := known[name]
		if !ok {
			return nil, fmt.Errorf("collect: source %q: unknown", name)
		}
		iv, err := time.ParseDuration(strings.TrimSpace(ds))
		if err != nil || iv <= 0 {
			return nil, fmt.Errorf("collect: source %q: bad interval %q", name, ds)
		}
		if seen[name] {
			return nil, fmt.Errorf("collect: source %q: listed twice", name)
		}
		seen[name] = true
		d.Interval = iv
		out = append(out, d)
	}
	return out, nil
}
