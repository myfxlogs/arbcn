import { useEffect, useState } from "react";

import type { Quote } from "./shared";

// useQuotes 秒级实时报价（D-056 Part B 下游）：EventSource 订阅 /quote/stream，
// onmessage 解析 JSON 更新 map（key = venue|symbol）。EventSource 原生自动重连——
// 断线由浏览器自动恢复，无需手动逻辑；组件卸载时 close 释放。只做展示，不喂策略。
export function useQuotes(): { quotes: Record<string, Quote> } {
  const [quotes, setQuotes] = useState<Record<string, Quote>>({});

  useEffect(() => {
    const es = new EventSource("/quote/stream");
    es.onmessage = (ev: MessageEvent) => {
      try {
        const q = JSON.parse(ev.data) as Quote;
        if (!q.venue || !q.symbol) return;
        setQuotes((prev) => ({ ...prev, [`${q.venue}|${q.symbol}`]: q }));
      } catch {
        // 解析失败跳过（脏帧不影响整体）
      }
    };
    return () => es.close();
  }, []);

  return { quotes };
}
