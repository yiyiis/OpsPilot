/*
=== AI Agent 概念：结构化数据库工具 ===

让 LLM 能够操作 MySQL 数据库——查询、插入、更新、删除数据。
这个工具展示了如何在 Agent 工具中集成数据库操作。

核心原理：
  - LLM 生成 SQL 语句和操作类型，通过工具参数传入
  - 工具内部使用 GORM 执行 SQL
  - 查询结果格式化为 JSON 返回给 LLM

本文件的角色：
  提供 MySQL CRUD 工具，LLM 可以自主决定何时需要查询数据库。
  包含交互式确认机制（stdin），防止 LLM 执行危险的 SQL 操作。

⚠️ 注意事项：
  - 交互式确认（stdin）在服务器环境中可能不适用
  - 使用 log.Fatal() 会在错误时终止整个进程

关联文件：
  - flow.go — 将此工具注册到 ReAct Agent
  - get_current_time.go — 更简单的工具示例（学习入口）
*/
package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// MysqlCrudInput MySQL 操作工具的输入参数
//
// 【AI 概念】Tool Schema 设计
// 每个字段的 jsonschema description 会作为工具参数描述发送给 LLM。
// LLM 根据这些描述来生成正确的参数值。
//
// 例如，当 LLM 需要查询用户表时，它会生成：
//   {"dsn": "user:pass@tcp(localhost:3306)/mydb", "sql": "SELECT * FROM users LIMIT 10", "operate_type": "query"}
type MysqlCrudInput struct {
	DSN         string `json:"dsn" jsonschema:"description=The Data Source Name for connecting to the MySQL database, including username, password, host, port, and database name"`
	SQL         string `json:"sql" jsonschema:"description=The SQL query to execute against the MySQL database"`
	OperateType string `json:"operate_type" jsonschema:"description=The type of SQL operation to perform: query, insert, update, or delete"`
}

// NewMysqlCrudTool 创建 MySQL 数据库操作工具
//
// 【AI 概念】安全工具设计
// 这个工具包含一个重要的安全机制：交互式确认。
// LLM 生成的 SQL 会在执行前展示给用户，用户确认后才执行。
// 这是为了防止 LLM 执行危险的 SQL（如 DROP TABLE、无条件 DELETE 等）。
//
// 工作流程：
//   1. LLM 决定需要查询数据库 → 生成 DSN、SQL、OperateType 参数
//   2. 工具使用 GORM 打开数据库连接
//   3. 在终端提示用户确认 SQL（y/n）
//   4. 用户确认后执行 SQL
//   5. 查询操作返回 JSON 格式的结果，其他操作返回空字符串
func NewMysqlCrudTool() tool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"mysql_crud", // 工具名
		"Execute SQL queries against the MySQL database and return results in JSON format. Use this tool when you need to query, insert, update or delete data from the database. The results will be formatted as JSON for easy parsing.",
		func(ctx context.Context, input *MysqlCrudInput, opts ...tool.Option) (output string, err error) {
			// 1. 使用 GORM 建立数据库连接
			db, err := gorm.Open(mysql.Open(input.DSN), &gorm.Config{})
			if err != nil {
				log.Fatal(err)
			}

			// 2. 交互式确认：在终端提示用户确认是否执行该 SQL
			// ⚠️ 注意：这种方式在服务器环境中（如 HTTP 服务）不适用，
			// 因为 stdin 可能不是连接到终端的
			scanner := bufio.NewScanner(os.Stdin)
			fmt.Print("\n请确定是否执行本sql(y/n): ", input.SQL)
			scanner.Scan()
			fmt.Println()
			nInput := scanner.Text()
			if nInput != "y" {
				return "用户取消执行sql", nil // 用户拒绝，返回取消信息给 LLM
			}

			// 3. 执行 SQL
			err = db.Exec(input.SQL).Error
			if err != nil {
				log.Fatal(err)
			}

			// 4. 如果是查询操作，返回 JSON 格式的结果
			if input.OperateType == "query" {
				var results []interface{}
				err = db.Raw(input.SQL).Scan(&results).Error
				if err != nil {
					log.Fatal(err)
				}
				resBytes, err := json.Marshal(results)
				return string(resBytes), err
			}
			// 非查询操作（insert/update/delete）返回空字符串
			return "", nil
		})
	if err != nil {
		log.Fatal(err)
	}
	return t
}
