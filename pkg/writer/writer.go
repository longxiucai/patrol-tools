// writer provides our stdout writers for promql query results
package writer

import (
	"bytes"

	"github.com/prometheus/common/model"
)

// Writer is our base interface for promql writers
// Defines Json and Csv writers
type Writer interface {
	Json() (bytes.Buffer, error)
	Csv(noHeaders bool) (bytes.Buffer, error)
	AppendJson(*bytes.Buffer) error
	AppendCsv(bool, *bytes.Buffer) error
}

func appendJson(resultbuf *bytes.Buffer, o []byte) error {
	// 如果resultbuf不为空，添加逗号
	if resultbuf.Len() > 0 {
		resultbuf.WriteString(",")
	}
	// 写入JSON数据到resultbuf
	_, err := resultbuf.Write(o)
	if err != nil {
		return err
	}
	return nil
}

func cvsAddTitle(rows *[][]string, name string, expr string) {
	var titleRow []string
	titleRow = append(titleRow, "name")
	titleRow = append(titleRow, "expr")
	*rows = append(*rows, titleRow)
	var titleValueRow []string
	titleValueRow = append(titleValueRow, name)
	titleValueRow = append(titleValueRow, expr)
	*rows = append(*rows, titleValueRow)
}

func cvsAddHeader(rows *[][]string, labels []model.LabelName) {
	var headerRow []string
	for _, k := range labels {
		headerRow = append(headerRow, string(k))
	}

	headerRow = append(headerRow, "value")
	headerRow = append(headerRow, "timestamp")
	*rows = append(*rows, headerRow)
}
