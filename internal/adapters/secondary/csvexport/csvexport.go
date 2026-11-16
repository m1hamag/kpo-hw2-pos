// Package csvexport выгружает итоги в CSV. Это secondary-адаптер для порта app.FinalsExporter
package csvexport

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"pos/internal/app"
)

var _ app.FinalsExporter = Exporter{}

// Exporter пишет ведомость с итогами в CSV. Разделитель точка с запятой,
// дробная часть через запятую, так файл сразу открывается в русском Excel
type Exporter struct{}

// New создаёт экспорт в CSV
func New() Exporter { return Exporter{} }

// Export пишет строку заголовка и по строке на студента
func (Exporter) Export(w io.Writer, finals []app.StudentFinal) error {
	header := []string{"№", "Студент"}
	if len(finals) > 0 {
		for _, l := range finals[0].Final.Lines() {
			header = append(header, fmt.Sprintf("%s (из %s)", l.Criterion.Name(), number(l.Criterion.MaxScore())))
		}
	}
	header = append(header, "Итог", "Оценка", "Правило")

	rows := [][]string{header}
	for _, f := range finals {
		row := []string{strconv.Itoa(int(f.Student.ID())), f.Student.Name()}
		for _, l := range f.Final.Lines() {
			cell := ""
			if l.Evaluation != nil {
				cell = number(l.Evaluation.Score().Points())
			}
			row = append(row, cell)
		}
		row = append(row,
			strings.Replace(strconv.FormatFloat(f.Final.Value(), 'f', 2, 64), ".", ",", 1),
			strconv.Itoa(f.Final.Rounded()),
			f.Final.Policy(),
		)
		rows = append(rows, row)
	}

	cw := csv.NewWriter(w)
	cw.Comma = ';'
	return cw.WriteAll(rows)
}

// number печатает число без лишних нулей и с запятой, например 7,5
func number(x float64) string {
	return strings.Replace(strconv.FormatFloat(x, 'f', -1, 64), ".", ",", 1)
}
