package clickhouse

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

// formatSQLWithArgs safely formats SQL with positional arguments into literal ClickHouse SQL for EXPLAIN plans.
func formatSQLWithArgs(sql string, args []interface{}) string {
	if len(args) == 0 {
		return sql
	}
	var sb strings.Builder
	argIdx := 0
	for i := 0; i < len(sql); i++ {
		if sql[i] == '?' && argIdx < len(args) {
			arg := args[argIdx]
			argIdx++
			switch v := arg.(type) {
			case string:
				sb.WriteString("'" + strings.ReplaceAll(v, "'", "\\'") + "'")
			case time.Time:
				sb.WriteString("'" + v.UTC().Format("2006-01-02 15:04:05") + "'")
			case int, int8, int16, int32, int64:
				sb.WriteString(fmt.Sprintf("%d", v))
			case uint, uint8, uint16, uint32, uint64:
				sb.WriteString(fmt.Sprintf("%d", v))
			case float32, float64:
				sb.WriteString(fmt.Sprintf("%f", v))
			case bool:
				if v {
					sb.WriteString("1")
				} else {
					sb.WriteString("0")
				}
			case [16]byte:
				sb.WriteString(fmt.Sprintf("unhex('%x')", v[:]))
			case [8]byte:
				sb.WriteString(fmt.Sprintf("unhex('%x')", v[:]))
			case []byte:
				sb.WriteString(fmt.Sprintf("unhex('%x')", v))
			default:
				sb.WriteString(fmt.Sprintf("'%v'", v))
			}
		} else {
			sb.WriteByte(sql[i])
		}
	}
	return sb.String()
}

// explainQuery runs EXPLAIN indexes = 1 on the given SQL and returns the formatted execution plan.
func explainQuery(ctx context.Context, conn clickhouse.Conn, sql string, args []interface{}) (string, error) {
	formattedSQL := formatSQLWithArgs(sql, args)
	explainSQL := "EXPLAIN indexes = 1 " + formattedSQL
	rows, err := conn.Query(ctx, explainSQL)
	if err != nil {
		explainSQL = "EXPLAIN " + formattedSQL
		rows, err = conn.Query(ctx, explainSQL)
		if err != nil {
			return "", fmt.Errorf("EXPLAIN failed: %w", err)
		}
	}
	defer rows.Close()

	var lines []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err == nil {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n"), nil
}

