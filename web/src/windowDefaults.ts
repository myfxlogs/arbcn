import { create } from "@bufbuild/protobuf";
import {
  FundingWindowStats,
  FundingWindowStatsSchema,
} from "./gen/arbcn/dashboard/v1/dashboard_pb";

// emptyWindowStats 窗口判据兜底（D-064）：overall 理论上服务端恒非空（RPC 恒返回
// toWindowStatsProto 结果），防御性兜底防 undefined 解构崩溃；类值 not = 不给出
// 可交易假象（宁缺毋滥 D-019）。
export const emptyWindowStats: FundingWindowStats = create(FundingWindowStatsSchema, {
  count: 0,
  min: 0,
  max: 0,
  mean: 0,
  positiveShare: 0,
  class: "not",
  note: "窗口判定不可用（无数据）",
});
