package result

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/longxiucai/patrol-tools/pkg/common"
	"github.com/longxiucai/patrol-tools/pkg/writer"

	"github.com/golang/glog"
	"github.com/prometheus/common/model"
)

type ResultWriter struct {
	ResultBuffer ResultBuf
	Options      ResultOptions
}

type ResultBuf struct {
	JsonResultBuf bytes.Buffer
	CsvResultBuf  bytes.Buffer
}

type ResultOptions struct {
	OutputPath       string
	OutputFilePrefix string
	OutPutTime       time.Time
}

func (rl ResultList) Write(op, format string, noheders bool) error {
	r := ResultWriter{
		Options: ResultOptions{
			OutputPath:       op,                      //  ./result/20231106-1530/
			OutputFilePrefix: common.OUTPUTFILEPREFIX, //promql_prometheus_data
			OutPutTime:       time.Now(),
		},
	}
	err := r.WriteResult(format, noheders, rl)
	if err != nil {
		return err
	}
	return nil
}

// WriteResults 方法接受整个 ResultList 并分派到不同的方法
func (w *ResultWriter) WriteResult(format string, noHeaders bool, rl ResultList) error {
	var excel common.ExcelFile
	var err error
	if format == "excel" {
		excel, err = common.NewExcelFile(filepath.Join(w.Options.OutputPath, w.Options.OutputFilePrefix+".xlsx"), w.Options.OutPutTime)
		if err != nil {
			glog.Error(err)
			return err
		}
	}

	for _, res := range rl {
		switch r := res.PromResult.(type) {
		case model.Vector:
			if err := w.WriteVectorResult(r, res.Rule, format, noHeaders, excel); err != nil {
				glog.Error(err)
				return err
			}
		case model.Matrix:
			if err := w.WriteMatrixResult(r, res.Rule, format, noHeaders); err != nil {
				glog.Error(err)
				return err
			}
		default:
			return fmt.Errorf("unsupported result type %v", r)
		}
	}

	// 存储文件
	switch format {
	case "excel":
		err := w.saveExcelTofile(excel)
		if err != nil {
			glog.Error(err)
			return err
		}
	case "json":
		_, err := w.saveJsonTofile()
		if err != nil {
			glog.Error(err)
			return err
		}
	case "csv":
		_, err := w.saveCsvTofile()
		if err != nil {
			glog.Error(err)
			return err
		}
	}
	return nil
}

// WriteVectorResult 方法用于处理 Vector 处理 VectorResult输出到适当的位置
func (w *ResultWriter) WriteVectorResult(res model.Vector, rule common.Rule, format string, noHeaders bool, excel common.ExcelFile) error {
	v := writer.VectorResult{
		Vector: res,
		Rule:   rule,
	}
	return writer.WriteVector(&v, format, noHeaders, excel, &w.ResultBuffer.JsonResultBuf, &w.ResultBuffer.CsvResultBuf)
}

// WriteMatrixResult方法用于处理 Matrix 处理 MatrixResult输出到适当的位置
func (w *ResultWriter) WriteMatrixResult(res model.Matrix, rule common.Rule, format string, noHeaders bool) error {
	m := writer.MatrixResult{
		Matrix: res,
		Rule:   rule,
	}
	return writer.WriteMatrix(&m, format, noHeaders, &w.ResultBuffer.JsonResultBuf, &w.ResultBuffer.CsvResultBuf)
}

func (w *ResultWriter) saveCsvTofile() (string, error) {
	csvStr := w.ResultBuffer.CsvResultBuf.String()
	fmt.Println(csvStr)
	// 将 buffer 中的内容写入文件
	csvFilePath := filepath.Join(w.Options.OutputPath, w.Options.OutputFilePrefix+fmt.Sprintf("-%02d-%02d-%02d",
		w.Options.OutPutTime.Hour(), w.Options.OutPutTime.Minute(), w.Options.OutPutTime.Second())+".csv")
	if err := ioutil.WriteFile(csvFilePath, []byte(csvStr), 0755); err != nil {
		return "", fmt.Errorf("error writing to file:%v", err)
	} else {
		glog.Infof("Data written to: %s\n", csvFilePath)
		return csvStr, nil
	}
}

func (w *ResultWriter) saveJsonTofile() (string, error) {
	jsonStr := fmt.Sprintf("[" + w.ResultBuffer.JsonResultBuf.String() + "]") // json数据全部组合完成之后 添加[]
	fmt.Println(jsonStr)
	// 将 buffer 中的内容写入文件

	jsonFilePath := filepath.Join(w.Options.OutputPath, w.Options.OutputFilePrefix+fmt.Sprintf("-%02d-%02d-%02d",
		w.Options.OutPutTime.Hour(), w.Options.OutPutTime.Minute(), w.Options.OutPutTime.Second())+".json")
	if err := ioutil.WriteFile(jsonFilePath, []byte(jsonStr), 0755); err != nil {
		return "", fmt.Errorf("error writing to file:%v", err)
	} else {
		glog.Infof("Data written to: %s\n", jsonFilePath)
		return jsonStr, nil
	}
}

func (w *ResultWriter) saveExcelTofile(excel common.ExcelFile) error {
	if err := excel.ExcelLize.SaveAs(excel.FullPath); err != nil { // excel文件保存到磁盘
		return fmt.Errorf("error writing to file:%v", err)
	} else {
		glog.Infof("Result save to: %s .Sheet name: %s\n", excel.FullPath, excel.SheetName)
		return nil
	}
}
