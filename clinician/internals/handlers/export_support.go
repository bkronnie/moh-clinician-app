package handlers

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type pdfObject struct {
	id   int
	data string
}

func writeCSVDownload(c *gin.Context, filename string, headers []string, rows [][]string) {
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Status(http.StatusOK)

	writer := csv.NewWriter(c.Writer)
	_ = writer.Write(headers)
	for _, row := range rows {
		_ = writer.Write(row)
	}
	writer.Flush()
}

func writeSimplePDFDownload(c *gin.Context, filename, title string, lines []string) {
	pdfBytes := buildSimplePDF(title, lines)
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

func buildSimplePDF(title string, lines []string) []byte {
	const linesPerPage = 44

	pages := paginatePDFLines(title, lines, linesPerPage)
	if len(pages) == 0 {
		pages = [][]string{{title, "", "No records found."}}
	}

	objects := []pdfObject{}
	nextID := 1

	catalogID := nextID
	nextID++
	pagesID := nextID
	nextID++
	fontID := nextID
	nextID++

	pageIDs := make([]int, 0, len(pages))
	contentIDs := make([]int, 0, len(pages))
	for range pages {
		pageIDs = append(pageIDs, nextID)
		nextID++
		contentIDs = append(contentIDs, nextID)
		nextID++
	}

	objects = append(objects, pdfObject{
		id:   fontID,
		data: "<< /Type /Font /Subtype /Type1 /BaseFont /Courier >>",
	})

	for index, pageLines := range pages {
		content := buildPDFContentStream(pageLines)
		objects = append(objects, pdfObject{
			id:   contentIDs[index],
			data: fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content),
		})
		objects = append(objects, pdfObject{
			id:   pageIDs[index],
			data: fmt.Sprintf("<< /Type /Page /Parent %d 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 %d 0 R >> >> /Contents %d 0 R >>", pagesID, fontID, contentIDs[index]),
		})
	}

	kids := make([]string, 0, len(pageIDs))
	for _, id := range pageIDs {
		kids = append(kids, fmt.Sprintf("%d 0 R", id))
	}

	objects = append(objects, pdfObject{
		id:   pagesID,
		data: fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), len(pageIDs)),
	})
	objects = append(objects, pdfObject{
		id:   catalogID,
		data: fmt.Sprintf("<< /Type /Catalog /Pages %d 0 R >>", pagesID),
	})

	sortPDFObjects(objects)

	var buffer bytes.Buffer
	buffer.WriteString("%PDF-1.4\n")

	offsets := make([]int, nextID)
	for _, object := range objects {
		offsets[object.id] = buffer.Len()
		buffer.WriteString(fmt.Sprintf("%d 0 obj\n%s\nendobj\n", object.id, object.data))
	}

	xrefOffset := buffer.Len()
	buffer.WriteString(fmt.Sprintf("xref\n0 %d\n", nextID))
	buffer.WriteString("0000000000 65535 f \n")
	for id := 1; id < nextID; id++ {
		buffer.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[id]))
	}
	buffer.WriteString(fmt.Sprintf("trailer\n<< /Size %d /Root %d 0 R >>\nstartxref\n%d\n%%%%EOF", nextID, catalogID, xrefOffset))

	return buffer.Bytes()
}

func paginatePDFLines(title string, lines []string, linesPerPage int) [][]string {
	if linesPerPage <= 0 {
		linesPerPage = 44
	}

	allLines := []string{title, ""}
	allLines = append(allLines, lines...)
	if len(allLines) == 0 {
		return nil
	}

	pages := [][]string{}
	current := []string{}
	for _, line := range allLines {
		wrapped := wrapPDFLine(line, 96)
		for _, segment := range wrapped {
			if len(current) >= linesPerPage {
				pages = append(pages, current)
				current = []string{}
			}
			current = append(current, segment)
		}
	}

	if len(current) > 0 {
		pages = append(pages, current)
	}

	return pages
}

func wrapPDFLine(line string, width int) []string {
	if width <= 0 {
		width = 96
	}

	clean := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(line, "\r", " "), "\n", " "))
	if clean == "" {
		return []string{""}
	}

	words := strings.Fields(clean)
	segments := []string{}
	current := words[0]
	for _, word := range words[1:] {
		if len(current)+1+len(word) > width {
			segments = append(segments, current)
			current = word
			continue
		}
		current += " " + word
	}
	segments = append(segments, current)
	return segments
}

func buildPDFContentStream(lines []string) string {
	var builder strings.Builder
	builder.WriteString("BT\n/F1 10 Tf\n40 760 Td\n14 TL\n")
	for index, line := range lines {
		if index > 0 {
			builder.WriteString("T*\n")
		}
		builder.WriteString("(")
		builder.WriteString(escapePDFText(line))
		builder.WriteString(") Tj\n")
	}
	builder.WriteString("ET")
	return builder.String()
}

func escapePDFText(value string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"(", "\\(",
		")", "\\)",
	)
	return replacer.Replace(value)
}

func sortPDFObjects(objects []pdfObject) {
	for i := 0; i < len(objects)-1; i++ {
		for j := i + 1; j < len(objects); j++ {
			if objects[j].id < objects[i].id {
				objects[i], objects[j] = objects[j], objects[i]
			}
		}
	}
}
