package di_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pos/internal/adapters/primary/console"
	"pos/internal/adapters/secondary/csvexport"
	"pos/internal/app"
	"pos/internal/di"
	"pos/internal/domain"
)

func today() time.Time { return time.Date(2026, time.November, 16, 12, 0, 0, 0, time.UTC) }

func TestContainer_ResolvesApp(t *testing.T) {
	c, err := di.NewContainer(di.Config{In: strings.NewReader(""), Out: io.Discard})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Invoke(func(a *console.App) {}); err != nil {
		t.Fatalf("контейнер не собрал консоль: %v", err)
	}
}

func TestContainer_BindsPortsToAdapters(t *testing.T) {
	c, _ := di.NewContainer(di.Config{})
	err := c.Invoke(func(exporter app.FinalsExporter, students domain.StudentRepository,
		rubric domain.RubricRepository, evaluations domain.EvaluationRepository, policies []domain.FinalPolicy) {
		if _, ok := exporter.(csvexport.Exporter); !ok {
			t.Errorf("порт выгрузки связан с %T", exporter)
		}
		if students == nil || rubric == nil || evaluations == nil {
			t.Error("репозитории не связаны")
		}
		if len(policies) != 2 {
			t.Errorf("правил %d", len(policies))
		}
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestContainer_SingleGradebook(t *testing.T) {
	c, _ := di.NewContainer(di.Config{})
	var first, second *app.Gradebook
	c.Invoke(func(g *app.Gradebook) { first = g })
	c.Invoke(func(g *app.Gradebook) { second = g })
	if first == nil || first != second {
		t.Error("фасад должен создаваться один раз")
	}
}

func TestContainer_RunsDemo(t *testing.T) {
	var out bytes.Buffer
	c, err := di.NewContainer(di.Config{In: console.DemoInput(&out), Out: &out, Now: today})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Invoke(func(a *console.App) error { return a.Run() }); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"№1   Иванов Иван          итог  7.60   оценка  8   выставлено 3 из 3",
		"1;Иванов Иван;8;5;6;7,60;8;по всем критериям, без оценки 0 баллов",
		"До встречи!",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("в выводе демо нет %q", want)
		}
	}
}

func TestContainer_KeepsDataInFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pos.json")
	input := "1\nИванов Иван\n4\nДЗ\n10\n1\n7\n1\n1\n9\n\n0\n"
	c, _ := di.NewContainer(di.Config{DataPath: path, In: strings.NewReader(input), Out: io.Discard, Now: today})
	if err := c.Invoke(func(a *console.App) error { return a.Run() }); err != nil {
		t.Fatal(err)
	}

	again, _ := di.NewContainer(di.Config{DataPath: path})
	err := again.Invoke(func(g *app.Gradebook) {
		f, err := g.Final(1)
		if err != nil {
			t.Fatal(err)
		}
		if f.Student.Name() != "Иванов Иван" || f.Final.Value() != 9 {
			t.Errorf("после перезапуска %s, итог %v", f.Student.Name(), f.Final.Value())
		}
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestContainer_BrokenFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pos.json")
	os.WriteFile(path, []byte(`{"criteria": [{"id": 1, "name": "ДЗ", "max_score": 10, "weight": 1}],
		"students": [{"id": 1, "name": "Иванов Иван"}],
		"evaluations": [{"student_id": 1, "criterion_id": 1, "score": 12, "date": "2026-11-16"}]}`), 0o644)
	c, err := di.NewContainer(di.Config{DataPath: path})
	if err != nil {
		t.Fatal(err)
	}
	err = c.Invoke(func(a *console.App) {})
	if !errors.Is(err, domain.ErrScoreOutOfRange) {
		t.Errorf("испорченный файл должен остановить запуск, получили %v", err)
	}
}

// spyExporter нужен, чтобы подменить адаптер через сам контейнер
type spyExporter struct{ calls int }

func (s *spyExporter) Export(io.Writer, []app.StudentFinal) error {
	s.calls++
	return nil
}

func TestContainer_ExporterReplacedWithStub(t *testing.T) {
	c, _ := di.NewContainer(di.Config{})
	spy := &spyExporter{}
	if err := c.Decorate(func(app.FinalsExporter) app.FinalsExporter { return spy }); err != nil {
		t.Fatal(err)
	}
	err := c.Invoke(func(g *app.Gradebook) {
		if _, err := g.ExportFinals(io.Discard); err != nil {
			t.Fatal(err)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if spy.calls != 1 {
		t.Errorf("заглушка вызвана %d раз", spy.calls)
	}
}
