// Package di собирает приложение в DI-контейнере dig. Это composition root,
// единственное место, где адаптеры связываются с портами
package di

import (
	"errors"
	"io"

	"go.uber.org/dig"

	"pos/internal/adapters/primary/console"
	"pos/internal/adapters/secondary/csvexport"
	"pos/internal/adapters/secondary/jsonstore"
	"pos/internal/app"
	"pos/internal/domain"
)

// Config задаёт то, что приходит снаружи при запуске
type Config struct {
	DataPath string    // файл с данными, пустой путь значит работу без сохранения
	In       io.Reader // откуда консоль читает команды
	Out      io.Writer // куда консоль пишет
	Now      app.Clock // часы, без них берётся time.Now
}

// NewContainer регистрирует все зависимости приложения
func NewContainer(cfg Config) (*dig.Container, error) {
	c := dig.New()
	err := errors.Join(
		// secondary-адаптеры за портами домена и приложения
		c.Provide(func() (*jsonstore.Store, error) { return jsonstore.Open(cfg.DataPath) }),
		c.Provide(func(s *jsonstore.Store) domain.StudentRepository { return s.Students() }),
		c.Provide(func(s *jsonstore.Store) domain.RubricRepository { return s.Rubric() }),
		c.Provide(func(s *jsonstore.Store) domain.EvaluationRepository { return s.Evaluations() }),
		c.Provide(csvexport.New, dig.As(new(app.FinalsExporter))),

		// правила подсчёта итога, первое работает по умолчанию
		c.Provide(Policies),
		c.Provide(func() app.Clock { return cfg.Now }),

		// фасад сценариев и консоль
		c.Provide(app.New),
		c.Provide(func(gb *app.Gradebook) *console.App { return console.New(gb, cfg.In, cfg.Out) }),
	)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// Policies возвращает правила подсчёта итога, из которых можно выбирать в консоли
func Policies() []domain.FinalPolicy {
	return []domain.FinalPolicy{domain.WeightedPolicy{}, domain.GradedOnlyPolicy{}}
}
