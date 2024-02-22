// writer provides our stdout writers for promql query results
package writer

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/longxiucai/patrol-tools/pkg/common"

	"github.com/golang/glog"
	"github.com/nalbury/promql-cli/pkg/util"
	"github.com/prometheus/common/model"
	"github.com/xuri/excelize/v2"
)

// VectorWriter extends the Writer interface by adding a Table method
// Use specifically for writing the results of vector queries
type VectorWriter interface {
	Writer
	Defult(noHeaders bool) (bytes.Buffer, error)
	Excel(common.ExcelFile) error
}

// VectorResult is wrapper of the prometheus model.Matrix type returned from vector queries
// Satisfies the VectorWriter interface
type VectorResult struct {
	common.Rule
	model.Vector
}

// Table returns the response from an vector query as a tab separated table
func (r *VectorResult) Defult(noHeaders bool) (bytes.Buffer, error) {
	var buf bytes.Buffer
	const padding = 4
	w := tabwriter.NewWriter(&buf, 0, 0, padding, ' ', 0)
	labels, err := util.UniqLabels(r.Vector)
	if err != nil {
		return buf, err
	}
	var titles []string
	titles = append(titles, "NAME")
	titles = append(titles, "EXPR")
	titleRow := strings.Join(titles, "\t")
	if _, err := fmt.Fprintln(w, titleRow); err != nil {
		return buf, err
	}

	var titleValues []string
	titleValues = append(titleValues, r.Name)
	titleValues = append(titleValues, r.Expr)
	titleValuesRow := strings.Join(titleValues, "\t")
	if _, err := fmt.Fprintln(w, titleValuesRow); err != nil {
		return buf, err
	}

	if !noHeaders {
		var headers []string
		for _, k := range labels {
			headers = append(headers, strings.ToUpper(string(k)))
		}
		headers = append(headers, "VALUE")
		headers = append(headers, "TIMESTAMP")
		headerRow := strings.Join(headers, "\t")
		if _, err := fmt.Fprintln(w, headerRow); err != nil {
			return buf, err
		}
	}

	for _, v := range r.Vector {
		data := make([]string, len(labels))
		for i, key := range labels {
			data[i] = string(v.Metric[key])
		}
		data = append(data, v.Value.String())
		data = append(data, v.Timestamp.Time().Format(time.RFC3339))
		row := strings.Join(data, "\t")
		if _, err := fmt.Fprintln(w, row); err != nil {
			return buf, err
		}
	}
	if err := w.Flush(); err != nil {
		return buf, err
	}
	return buf, nil
}

// Json returns the response from an vector query as json
func (r *VectorResult) Json() (bytes.Buffer, error) {
	var buf bytes.Buffer
	o, err := json.Marshal(r)
	if err != nil {
		return buf, err
	}
	buf.Write(o)
	return buf, nil
}
func (r *VectorResult) AppendJson(jsonResultBuf *bytes.Buffer) error {
	o, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return appendJson(jsonResultBuf, o)
}

func (r *VectorResult) AppendCsv(noHeaders bool, resultbuf *bytes.Buffer) error {
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

// Csv returns the response from an vector query as a csv
func (r *VectorResult) Csv(noHeaders bool) (bytes.Buffer, error) {
	var (
		buf  bytes.Buffer
		rows [][]string
	)
	w := csv.NewWriter(&buf)
	labels, err := util.UniqLabels(r.Vector)
	if err != nil {
		return buf, err
	}

	cvsAddTitle(&rows, r.Name, r.Expr)

	if !noHeaders {
		cvsAddHeader(&rows, labels)
	}

	for _, v := range r.Vector {
		row := make([]string, len(labels))
		for i, key := range labels {
			row[i] = string(v.Metric[key])
		}
		row = append(row, v.Value.String())
		row = append(row, v.Timestamp.Time().Format(time.RFC3339))
		rows = append(rows, row)
	}
	if err := w.WriteAll(rows); err != nil {
		return buf, err
	}
	return buf, nil
}

// 获取需要添加的表头
func getNeedToAddExcelHeader(old, new []model.LabelName) []model.LabelName {
	oldLabelMap := make(map[model.LabelName]struct{})
	for _, label := range old {
		oldLabelMap[label] = struct{}{}
	}
	// 查找在新的标签名称中但不在旧的标签名称中的标签
	var newLabelsNotInOld []model.LabelName
	for _, label := range new {
		if _, exists := oldLabelMap[label]; !exists {
			newLabelsNotInOld = append(newLabelsNotInOld, label)
		}
	}
	// 判断value time
	if _, exists := oldLabelMap["VALUE"]; !exists {
		newLabelsNotInOld = append(newLabelsNotInOld, "VALUE")
	}
	if _, exists := oldLabelMap["TIMESTAMP"]; !exists {
		newLabelsNotInOld = append(newLabelsNotInOld, "TIMESTAMP")
	}
	if _, exists := oldLabelMap["RULE"]; !exists {
		newLabelsNotInOld = append(newLabelsNotInOld, "RULE")
	}

	return newLabelsNotInOld
}

func (r *VectorResult) Excel(excel common.ExcelFile) error {
	// 获取新数据标题行
	labelsInResult, err := util.UniqLabels(r.Vector)
	if err != nil {
		return err
	}
	sheetName := excel.SheetName

	// 判断是否有sheet，没有则创建sheet
	sheetExists := false
	sheetList := excel.ExcelLize.GetSheetList()
	for _, sheet := range sheetList {
		if sheet == sheetName {
			sheetExists = true
		}
	}
	if !sheetExists {
		if _, err := excel.ExcelLize.NewSheet(sheetName); err != nil {
			return err
		}

	}
	// 读取表
	rows, err := excel.ExcelLize.GetRows(sheetName)
	if err != nil {
		return err
	}

	// 获取已存在的表头
	var oldHeader []model.LabelName
	if len(rows) >= 1 {
		for _, colCell := range rows[0] {
			oldHeader = append(oldHeader, model.LabelName(colCell))
		}
	}

	// 新数据的表头 在excel表中找不到 ,则需要插在最前面
	needToAddExcel := getNeedToAddExcelHeader(oldHeader, labelsInResult)
	if len(needToAddExcel) > 0 {
		if err = excel.ExcelLize.InsertCols(sheetName, "A", len(needToAddExcel)); err != nil {
			return err
		}
		if err = excel.ExcelLize.SetSheetRow(sheetName, "A1", &needToAddExcel); err != nil {
			return err
		}
		// 设置列宽 如何auto？？
		// var addColName string
		// if addColName, err = excelize.ColumnNumberToName(len(needToAddExcel)); err != nil {
		// 	return err
		// }
		// if err = excel.ExcelLize.SetColWidth(sheetName, "A", addColName, 20); err != nil {
		// 	return err
		// }
	}

	// 将键值数据填入对应列。读取表头，再查找metrics，有则填入
	newHeader := append(needToAddExcel, oldHeader...)
	nextRowIndex := len(rows) + 1 //下一个空的行

	mergeColIndex := len(newHeader) // RULE列
	mergeRowIndex := nextRowIndex + 1

	for _, metrics := range r.Vector {
		for colIndex, headerValue := range newHeader {
			colName, err := excelize.ColumnNumberToName(colIndex + 1) //根据列数字转换成字母
			if err != nil {
				glog.Error(err)
				return err
			}
			cellName := colName + fmt.Sprintf("%d", nextRowIndex+1) //获取单元格具体位置
			cellValue, exists := metrics.Metric[headerValue]
			if exists {
				if err = excel.ExcelLize.SetCellValue(sheetName, cellName, cellValue); err != nil {
					return err
				}
			}

			switch headerValue {
			case "VALUE":
				if err = excel.ExcelLize.SetCellValue(sheetName, cellName, fmt.Sprintf("%.2f", metrics.Value)); err != nil {
					return err
				}
			case "TIMESTAMP":
				if err = excel.ExcelLize.SetCellValue(sheetName, cellName, metrics.Timestamp.Time().Format(time.RFC3339)); err != nil {
					return err
				}
			}
		}
		nextRowIndex++
	}

	// 合并最后一列，填入Name Expr
	mergeColName, err := excelize.ColumnNumberToName(mergeColIndex)
	if err != nil {
		glog.Error(err)
	}
	mergeCellName1 := mergeColName + fmt.Sprintf("%d", mergeRowIndex)
	mergeCellName2 := mergeColName + fmt.Sprintf("%d", nextRowIndex)
	excel.ExcelLize.MergeCell(sheetName, mergeCellName1, mergeCellName2)
	if err != nil {
		glog.Error(err)
	}
	if err = excel.ExcelLize.SetCellValue(sheetName, mergeCellName1, fmt.Sprintf("%s\n%s", r.Name, r.Expr)); err != nil {
		return err
	}

	// 设置合并后的单元格样式以启用自动换行、居中
	style, err := excel.ExcelLize.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			WrapText: true,
			Vertical: "center",
		},
	})
	if err != nil {
		return err
	}
	excel.ExcelLize.SetCellStyle(sheetName, mergeCellName1, mergeCellName2, style)
	if err != nil {
		return err
	}
	excel.ExcelLize.SetColWidth(sheetName, mergeColName, mergeColName, 40)
	if err != nil {
		return err
	}
	if err := excel.ExcelLize.SaveAs(excel.FullPath); err != nil { // excel文件保存到磁盘
		return err
	}

	return nil
}

// WriteVector writes out the results of the query to an
// output buffer and prints it to stdout
func WriteVector(v VectorWriter, format string, noHeaders bool, excel common.ExcelFile, jsonResultBuf *bytes.Buffer, csvResultBuf *bytes.Buffer) error {
	var (
		buf bytes.Buffer
		err error
	)
	switch format {
	case "json":
		err = v.AppendJson(jsonResultBuf)
		if err != nil {
			return err
		}
	case "csv":
		err = v.AppendCsv(noHeaders, csvResultBuf)
		if err != nil {
			return err
		}
	case "excel":
		err = v.Excel(excel)
		if err != nil {
			return err
		}
	default:
		buf, err = v.Defult(noHeaders)
		if err != nil {
			return err
		}
		fmt.Println(buf.String())
	}
	return nil
}
