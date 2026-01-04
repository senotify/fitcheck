package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
)

// QueryAnalyzer provides tools for analyzing and optimizing database queries
type QueryAnalyzer struct {
	db *sql.DB
}

// NewQueryAnalyzer creates a new query analyzer
func NewQueryAnalyzer(db *sql.DB) *QueryAnalyzer {
	return &QueryAnalyzer{db: db}
}

// ExplainQuery runs EXPLAIN ANALYZE on a query and returns the execution plan
func (qa *QueryAnalyzer) ExplainQuery(ctx context.Context, query string, args ...interface{}) (string, error) {
	explainQuery := "EXPLAIN ANALYZE " + query
	
	rows, err := qa.db.QueryContext(ctx, explainQuery, args...)
	if err != nil {
		return "", fmt.Errorf("failed to explain query: %w", err)
	}
	defer rows.Close()

	var plan strings.Builder
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return "", fmt.Errorf("failed to scan explain result: %w", err)
		}
		plan.WriteString(line)
		plan.WriteString("\n")
	}

	if err = rows.Err(); err != nil {
		return "", fmt.Errorf("error reading explain results: %w", err)
	}

	return plan.String(), nil
}

// VerifyIndexUsage checks if a query is using the expected indexes
func (qa *QueryAnalyzer) VerifyIndexUsage(ctx context.Context, query string, expectedIndexes []string, args ...interface{}) (bool, string, error) {
	plan, err := qa.ExplainQuery(ctx, query, args...)
	if err != nil {
		return false, "", err
	}

	// Check if any of the expected indexes are mentioned in the plan
	for _, index := range expectedIndexes {
		if strings.Contains(plan, index) {
			log.Printf("Query is using index: %s", index)
			return true, plan, nil
		}
	}

	// Check for sequential scan which indicates no index usage
	if strings.Contains(plan, "Seq Scan") {
		log.Printf("WARNING: Query is using sequential scan instead of index")
		return false, plan, nil
	}

	return false, plan, nil
}

// LogQueryPerformance logs the execution time and plan for a query
func (qa *QueryAnalyzer) LogQueryPerformance(ctx context.Context, queryName string, query string, args ...interface{}) {
	plan, err := qa.ExplainQuery(ctx, query, args...)
	if err != nil {
		log.Printf("Failed to analyze query %s: %v", queryName, err)
		return
	}

	log.Printf("Query Performance Analysis for %s:\n%s", queryName, plan)
}
