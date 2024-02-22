// writer provides our stdout writers for promql query results
package writer

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/longxiucai/patrol-tools/pkg/common"

	"github.com/guptarohit/asciigraph"
	"github.com/nalbury/promql-cli/pkg/util"
	"github.com/prometheus/common/model"
)

// MatrixWriter extends the Writer interface by adding a Graph method
// Used specifically for writing the results of range queries
type MatrixWriter interface {
	Writer
	Defult(dim util.TermDimensions) (bytes.Buffer, error)
}

// MatrixResult is wrapper of the prometheus model.Matrix type returned from range queries
// Satisfies the MatrixWriter interface
type MatrixResult struct {
	common.Rule
	model.Matrix
}

// Graph returns an ascii graph using https://github.com/guptarohit/asciigraph
func (r *MatrixResult) Defult(dim util.TermDimensions) (bytes.Buffer, error) {
	var buf bytes.Buffer

	termHeightOpt := asciigraph.Height(dim.Height / 5)
	termWidthOpt := asciigraph.Width(dim.Width - 8)

	for _, m := range r.Matrix {
		var (
			data         []float64
			start        string
			end          string
			borderLength int
		)

		for _, v := range m.Values {
			data = append(data, float64(v.Value))
		}

		start = m.Values[0].Timestamp.Time().Format(time.Stamp)
		end = m.Values[(len(m.Values) - 1)].Timestamp.Time().Format(time.Stamp)

		timeRange := start + " -> " + end

		// Generate the graph boxed to our terminal size
		graph := asciigraph.Plot(data, termHeightOpt, termWidthOpt)

		// Create title for each graph
		nameHeader := "# Name: " + r.Name
		exprHeader := "# EXPR: " + r.Expr
		// Create our header for each graph
		// # TIME_RANGE: Sep 27 09:08:09 -> Sep 27 09:18:09
		timeRangeHeader := "# TIME_RANGE: " + timeRange
		// # METRIC: {instance="10.202.38.101:6443"}
		metricHeader := "# METRIC: " + m.Metric.String()
		// Truncate the metric header to the term width - 2
		// This ensures that long metric headers don't overflow onto a new line
		if len(metricHeader) > (dim.Width - 2) {
			metricHeader = metricHeader[:(dim.Width - 2)]
		}

		// Determine the longest header string and set the border (######) to it's length + 2
		// Add spacing to the shortest header
		maxlenth := math.Max(math.Max(float64(len(metricHeader)), float64(len(metricHeader))), math.Max(float64(len(nameHeader)), float64(len(exprHeader))))
		borderLength = int(maxlenth) + 2
		metricHeader = metricHeader + strings.Repeat(" ", (int(maxlenth)-len(metricHeader)))
		nameHeader = nameHeader + strings.Repeat(" ", (int(maxlenth)-len(nameHeader)))
		exprHeader = exprHeader + strings.Repeat(" ", (int(maxlenth)-len(exprHeader)))
		timeRangeHeader = timeRangeHeader + strings.Repeat(" ", (int(maxlenth)-len(timeRangeHeader)))

		// Create the border of '#'
		border := strings.Repeat("#", borderLength)
		// Write out
		if _, err := fmt.Fprintf(&buf, "\n%s\n", border); err != nil {
			return buf, err
		}
		if _, err := fmt.Fprintf(&buf, "%s #\n", nameHeader); err != nil {
			return buf, err
		}
		if _, err := fmt.Fprintf(&buf, "%s #\n", exprHeader); err != nil {
			return buf, err
		}
		if _, err := fmt.Fprintf(&buf, "%s #\n", timeRangeHeader); err != nil {
			return buf, err
		}
		if _, err := fmt.Fprintf(&buf, "%s #\n", metricHeader); err != nil {
			return buf, err
		}
		if _, err := fmt.Fprintf(&buf, "%s\n", border); err != nil {
			return buf, err
		}
		if _, err := fmt.Fprintf(&buf, "%s\n", graph); err != nil {
			return buf, err
		}
	}
	return buf, nil
}
func (r *MatrixResult) AppendJson(resultbuf *bytes.Buffer) error {
	o, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return appendJson(resultbuf, o)
}

// Json returns the response from a range query as json
func (r *MatrixResult) Json() (bytes.Buffer, error) {
	var buf bytes.Buffer
	o, err := json.Marshal(r)
	if err != nil {
		return buf, err
	}
	buf.Write(o)
	return buf, nil
}
func (r *MatrixResult) AppendCsv(noHeaders bool, resultbuf *bytes.Buffer) error {
	res, err := r.Csv(noHeaders)
	if err != nil {
		return err
	}
	// 将res写入resultbuf
	_, err = resultbuf.Write(res.Bytes())
	if err != nil {
		return err
	}
	return nil
}

// Csv returns the response from a range query as a csv
func (r *MatrixResult) Csv(noHeaders bool) (bytes.Buffer, error) {
	var (
		buf  bytes.Buffer
		rows [][]string
	)
	w := csv.NewWriter(&buf)
	labels, err := util.UniqLabels(r.Matrix)
	if err != nil {
		return buf, err
	}
	cvsAddTitle(&rows, r.Name, r.Expr)

	if !noHeaders {
		cvsAddHeader(&rows, labels)
	}

	for _, m := range r.Matrix {
		for _, v := range m.Values {
			row := make([]string, len(labels))
			for i, key := range labels {
				row[i] = string(m.Metric[key])
			}
			row = append(row, v.Value.String())
			row = append(row, v.Timestamp.Time().Format(time.RFC3339))
			rows = append(rows, row)
		}
	}
	if err := w.WriteAll(rows); err != nil {
		return buf, err
	}
	return buf, nil
}

// WriteMatrix writes out the results of the query to an
// output buffer and prints it to stdout
func WriteMatrix(m MatrixWriter, format string, noHeaders bool, jsonResultBuf *bytes.Buffer, csvResultBuf *bytes.Buffer) error {
	var (
		buf bytes.Buffer
		err error
	)
	switch format {
	case "json":
		err = m.AppendJson(jsonResultBuf)
		if err != nil {
			return err
		}
	case "csv":
		err = m.AppendCsv(noHeaders, csvResultBuf)
		if err != nil {
			return err
		}
	default:
		dim, err := util.TerminalSize()
		if err != nil {
			return err
		}
		buf, err = m.Defult(dim)
		if err != nil {
			return err
		}
		fmt.Println(buf.String())
	}
	return nil
}
