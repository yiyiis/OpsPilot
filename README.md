# OpsPilot 🚀

**AI 驱动的智能运维助手** — 基于大语言模型的 OnCall 值班辅助系统

OpsPilot 是一个面向运维团队的 AI 助手，结合 RAG 知识问答和 Plan-Execute-Replan 告警分析，帮助运维人员快速定位和解决线上故障。

## ✨ 功能特性

- **智能知识问答** — 上传运维文档，通过自然语言查询处理手册（RAG 检索增强生成）
- **告警自动分析** — 输入告警描述，AI 自动规划排查步骤、调用工具、生成诊断报告
- **多工具协作** — 查询日志（MCP）、联网搜索（智谱 MCP）、Prometheus 告警、MySQL 数据库、内部文档
- **流式对话** — SSE 实时推送，打字机效果，支持多轮对话
- **文件上传** — 拖拽上传 .md/.txt 文件，自动分块、向量化、入库

## 🏗️ 系统架构

```text
用户浏览器 (:8080)
    │ SSE / HTTP
    ▼
Gin HTTP 服务 (:6872)
    │
    ├── /chat          → RAG Chat Agent（Eino DAG 图编排）
    ├── /chat_stream   → RAG Chat Agent（SSE 流式）
    ├── /upload        → Knowledge Index Pipeline
    └── /ai_ops        → Plan-Execute-Replan Agent
    │
    ▼
共享基础设施
    ├── Milvus 向量数据库（知识存储与检索）
    ├── 智谱 GLM-5.1 LLM（Quick + Think 双模型）
    ├── Gemini Embedding（文本向量化）
    └── MCP（日志查询 + 联网搜索 + 网页读取）/ Prometheus / MySQL
```

## 🛠️ 技术栈

| 组件 | 技术选型 | 说明 |
| ---- | -------- | ---- |
| 后端框架 | Gin | HTTP 路由、中间件、配置管理 |
| Agent 框架 | CloudWeGo Eino | 图编排、ReAct Agent、工具调用 |
| LLM | 智谱 GLM-5.1（Quick + Think） | Quick 用于对话/执行，Think 用于规划/重规划 |
| Embedding | Google Gemini embedding-2 | 文本向量化（3072维） |
| 向量数据库 | Milvus v2.5 | 知识存储与语义检索 |
| 工具协议 | MCP (Model Context Protocol) | 多服务器：日志查询 + 智谱联网搜索 + 网页读取 |
| 前端 | 原生 JS + SSE | Gemini 风格聊天界面，零构建 |
| 语言 | Go 1.25+ | 后端 |

## 📋 前置条件

- **Go 1.25+** — 后端运行环境
- **Docker & Docker Compose** — 启动 Milvus 向量数据库
- **智谱 API Key** — 用于 LLM 和 MCP 服务（联网搜索、网页读取）
- **Google Gemini API Key** — 用于 Embedding 模型（需要代理）
- **HTTP 代理** — Gemini API 需要翻墙访问

## 🚀 快速启动

### Step 1: 启动 Milvus 向量数据库

```bash
cd manifest/docker
docker-compose up -d
```

启动后包含 4 个服务：

- **Milvus** — 向量数据库，端口 `19530`
- **Attu** — Milvus Web 管理界面，端口 `8000`
- **etcd + MinIO** — Milvus 依赖的元数据存储和对象存储

### Step 2: 配置 API Key

API Key 通过环境变量管理，**不要写在 config.yaml 中**。

```bash
# 复制模板
cp .env.example .env

# 编辑 .env，填入你的真实 API Key
vim .env
```

`.env` 文件内容：

```bash
# 智谱 GLM API Key（Think + Quick 模型共用，以及 MCP 联网搜索/网页读取）
GLM_THINK_CHAT_MODEL_API_KEY=你的智谱APIKey
GLM_QUICK_CHAT_MODEL_API_KEY=你的智谱APIKey

# Google Gemini Embedding API Key（需要 HTTPS_PROXY 代理）
GEMINI_EMBEDDING_MODEL_API_KEY=你的GeminiAPIKey
```

> ⚠️ `.env` 已在 `.gitignore` 中，不会被提交到 Git。`.env.example` 作为模板提交。

**配置优先级**：环境变量（含 `.env`） > `config.yaml` > 默认值

### Step 3: 灌入知识数据（必须先于后端启动）

```bash
# 删除旧 collection（如果有的话）
curl -s -X POST http://localhost:19530/v2/vectordb/collections/drop \
  -H "Content-Type: application/json" -d '{"collectionName":"biz"}'

# 灌入数据（加代理，Gemini API 需要翻墙）
HTTPS_PROXY=http://你的代理地址:端口 go run internal/ai/cmd/knowledge_cmd/main.go
```

> ⚠️ **重要**：必须先灌入数据，再启动后端。否则后端会以错误的 schema 自动创建 collection。

### Step 4: 启动日志 MCP Server

```bash
go run internal/ai/mcp/log_reader_server/main.go
```

日志查询 MCP 服务运行在 `http://localhost:3001/sse`

### Step 5: 启动后端

```bash
# 加代理（Gemini API 需要翻墙）
HTTPS_PROXY=http://你的代理地址:端口 go run main.go
```

后端运行在 `http://localhost:6872`

### Step 6: 启动前端

```bash
cd OpsPilotFrontend
bash start.sh
```

前端运行在 `http://localhost:8080`（需要 Python3 或 Node.js）

### Step 7: 访问应用

打开浏览器访问 <http://localhost:8080>

## 📡 API 接口

| 端点 | 方法 | 说明 |
| ---- | ---- | ---- |
| `/api/chat` | POST | 同步 RAG 对话，返回完整回答 |
| `/api/chat_stream` | POST | SSE 流式 RAG 对话，逐 token 推送 |
| `/api/upload` | POST (multipart) | 上传文件到知识库，自动索引 |
| `/api/ai_ops` | POST | AI Ops 告警分析（Plan-Execute-Replan） |

### 请求示例

**对话接口：**

```bash
curl -X POST http://localhost:6872/api/chat_stream \
  -H "Content-Type: application/json" \
  -d '{"Id": "session-1", "Question": "服务出现 panic 怎么排查？"}'
```

**文件上传：**

```bash
curl -X POST http://localhost:6872/api/upload \
  -F "file=@告警处理手册.md"
```

## 📁 项目结构

```text
OpsPilot/
├── main.go                                      # HTTP 服务入口（Gin, :6872）
├── go.mod                                       # Go 模块定义
├── .env                                         # 环境变量（API Key，不提交 Git）
├── .env.example                                 # 环境变量模板
├── api/                                         # API 接口定义
├── cmd/
│   └── debug_indexer/                           # 索引调试工具
├── internal/
│   ├── ai/
│   │   ├── agent/
│   │   │   ├── chat_pipeline/                   # RAG Chat Agent（Eino DAG 图）
│   │   │   ├── plan_execute_replan/             # Plan-Execute-Replan Agent
│   │   │   └── knowledge_index_pipeline/        # 知识索引 Pipeline
│   │   ├── cmd/                                 # AI 独立命令
│   │   │   ├── ai_ops_cmd/                      # 告警分析命令
│   │   │   ├── chat_cmd/                        # 聊天命令
│   │   │   ├── knowledge_cmd/                   # 知识索引命令
│   │   │   ├── llm_tool_cmd/                    # LLM 工具测试
│   │   │   └── recall_cmd/                      # 召回测试
│   │   ├── tools/                               # Agent 工具集（MCP/告警/SQL/时间/文档）
│   │   ├── mcp/log_reader_server/               # 本地日志 MCP Server（SSE :3001）
│   │   ├── models/                              # LLM 模型工厂（智谱 GLM Quick + Think）
│   │   ├── embedder/                            # Embedding 封装（Gemini）
│   │   ├── indexer/                             # Milvus 索引器（FloatVector, COSINE）
│   │   ├── retriever/                           # Milvus 检索器
│   │   └── loader/                              # 文件加载器
│   ├── controller/chat/                         # HTTP Controller（路由处理）
│   └── logic/sse/                               # SSE 流式输出服务
├── utility/
│   ├── client/                                  # Milvus 客户端初始化
│   ├── config/                                  # Viper 配置加载 + .env 支持
│   ├── common/                                  # 共享常量
│   ├── mem/                                     # 对话记忆管理（内存滑动窗口）
│   ├── middleware/                              # CORS + 响应中间件
│   └── log_call_back/                           # 日志回调
├── manifest/
│   ├── config/config.yaml                       # 主配置文件（不含 API Key）
│   └── docker/                                  # Docker Compose（Milvus 集群）
├── knowledge/                                   # 知识库文档目录
├── OpsPilotFrontend/                            # 前端（原生 JS + SSE）
└── docs/
    ├── ai-agent-architecture-guide.md           # AI Agent 架构学习指南
    ├── superpowers/specs/                       # 设计规格文档
    └── swagger/                                 # Swagger API 文档
```

## 📖 学习资源

如果你想深入了解本项目的 AI Agent 技术实现，请阅读：

- [AI Agent 架构指南](docs/ai-agent-architecture-guide.md) — 7 个核心概念 + 数据流图 + 推荐阅读顺序
- 各模块目录下的 `README.md` — 模块级概念说明
- 代码文件中的中文注释 — 逐函数讲解 AI Agent 概念

## 📄 License

MIT
