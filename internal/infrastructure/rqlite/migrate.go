package rqlite

import (
	"context"
	"fmt"
	"slices"
)

const dropWatchItems = `DROP TABLE IF EXISTS watch_items`

// migrateSchemaIfNeeded は旧スキーマ（auction_id 列）からの初回移行時のみ watch_items を DROP する。
func migrateSchemaIfNeeded(ctx context.Context, h HTTPClient) error {
	cols, err := tableColumns(ctx, h, "watch_items")
	if err != nil {
		return fmt.Errorf("inspect watch_items schema: %w", err)
	}
	if len(cols) == 0 || slices.Contains(cols, "market") {
		return nil
	}
	_, err = h.ExecuteSingle(ctx, dropWatchItems)
	if err != nil {
		return fmt.Errorf("drop legacy watch_items: %w", err)
	}
	return nil
}

func tableColumns(ctx context.Context, h HTTPClient, table string) ([]string, error) {
	resp, err := h.QuerySingle(ctx, fmt.Sprintf(`SELECT name FROM pragma_table_info('%s')`, table))
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}
	results := resp.GetQueryResults()
	if len(results) == 0 {
		return nil, nil
	}
	var cols []string
	for _, row := range results[0].Values {
		if len(row) > 0 {
			cols = append(cols, toString(row[0]))
		}
	}
	return cols, nil
}
