package controllers

import (
	"encoding/csv"
	"fmt"

	"github.com/gin-gonic/gin"
)

// writeCSV streams rows as a CSV download response (docs/*.md export endpoints).
func writeCSV(c *gin.Context, filename string, header []string, rows [][]string) {
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w := csv.NewWriter(c.Writer)
	defer w.Flush()
	_ = w.Write(header)
	for _, row := range rows {
		_ = w.Write(row)
	}
}
