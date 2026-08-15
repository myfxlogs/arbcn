#!/usr/bin/env bash
# 文件规模门禁（docs/design/02-monitor-architecture.md §11 / AGENTS.md §7.3 C）：
# Go/TS 源文件 >450 行硬阻断（exit 1）、>300 行软警告；豁免 proto 生成代码与
# 测试文件（dialogue #28 裁决：生成代码豁免）。由 pre-commit 调用，也可单独跑。
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
SOFT=300
HARD=450

# exempt：proto 生成代码（internal/dashboard/gen/、web/src/gen/）与 *_test.go。
exempt() {
  case "$1" in
    internal/dashboard/gen/*|web/src/gen/*|*_test.go) return 0 ;;
  esac
  return 1
}

hard=0 soft=0
while IFS= read -r f; do
  rel="${f#"$ROOT"/}"
  exempt "$rel" && continue
  n="$(wc -l < "$f")"
  if [ "$n" -gt "$HARD" ]; then
    echo "❌ $rel: $n 行（> $HARD 硬阻断）"
    hard=1
  elif [ "$n" -gt "$SOFT" ]; then
    echo "⚠️  $rel: $n 行（> $SOFT 警告）"
    soft=1
  fi
done < <(find "$ROOT" -type f \( -name '*.go' -o -name '*.ts' -o -name '*.tsx' \) \
  -not -path '*/node_modules/*' -not -path '*/.git/*')

if [ "$hard" -eq 1 ]; then
  echo "❌ check-lines: 存在 >$HARD 行的源文件，提交被阻断"
  exit 1
fi
if [ "$soft" -eq 1 ]; then
  echo "⚠️  check-lines: 存在 >$SOFT 行的源文件（软警告，建议拆分）"
fi
exit 0
