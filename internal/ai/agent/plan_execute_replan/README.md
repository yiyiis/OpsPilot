# Plan-Execute-Replan Agent 模块

## 概述

本模块实现了 OpsPilot 的 **AI Ops 告警分析 Agent**，采用 Plan-Execute-Replan（规划-执行-重规划）模式。

核心思路：面对复杂的运维问题，先制定分步计划 → 逐步执行（每步可调用工具）→ 根据执行结果决定是否需要调整计划 → 循环直到任务完成。

## 核心概念

### Plan-Execute-Replan 模式

比 ReAct 更结构化的 Agent 模式。ReAct 是"想到哪做到哪"，Plan-Execute-Replan 是"先制定完整计划，再按计划执行，根据结果动态调整"。

**三阶段分工：**

```
用户问题 → Planner → Executor → Replanner → (循环或输出)
              ↑                              │
              └──────────────────────────────┘
                   (如需调整计划)
```

### 模型选择策略

本项目使用两种不同的 LLM 模型，各司其职：

| 角色 | 模型 | 原因 |
|------|------|------|
| **Planner（规划器）** | GLM-5.1 Think（智谱） | 需要深度推理来制定合理的执行计划 |
| **Executor（执行器）** | GLM-5.1 Quick（智谱） | 需要快速响应，逐个执行具体步骤 |
| **Replanner（重规划器）** | GLM-5.1 Think（智谱） | 需要深度推理来评估执行结果并调整计划 |

**关键洞察：** 规划需要"慢思考"（Think 模型会展示推理过程），执行需要"快反应"（Quick 模型直接给出答案）。

## 文件清单

| 文件 | 职责 | 行数 |
|------|------|------|
| `plan_execute_replan.go` | 顶层编排：组装三阶段 Agent，运行事件流，收集结果 | 57 |
| `planner.go` | 创建规划器，使用 Think 模型 | 19 |
| `executor.go` | 创建执行器，绑定 4 个工具，使用 Quick 模型 | 39 |
| `replan.go` | 创建重规划器，使用 Think 模型 | 19 |

## 执行器可用工具

| 工具 | 用途 |
|------|------|
| `query_log` (MCP) | 通过 MCP 协议查询日志 |
| `query_prometheus_alerts` | 查询 Prometheus 活跃告警 |
| `query_internal_docs` | RAG 检索内部运维文档 |
| `get_current_time` | 获取当前时间 |

## 循环参数

- **最大迭代次数：** 20 次（`MaxIterations: 20`）
- **执行器内部最大迭代：** 999999（Executor 的 MaxIterations，确保执行器不会提前终止）
