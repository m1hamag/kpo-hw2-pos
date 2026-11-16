package domain

// FinalPolicy задаёт правило подсчёта итога (паттерн "Стратегия").
// Compute получает критерии в порядке рубрикатора и возвращает итог от 0 до 10
type FinalPolicy interface {
	Name() string
	Compute(lines []FinalLine) float64
}

// WeightedPolicy считает взвешенное среднее по всем критериям.
// Критерий без оценки даёт 0 баллов
type WeightedPolicy struct{}

// Name возвращает название правила
func (WeightedPolicy) Name() string {
	return "по всем критериям, без оценки 0 баллов"
}

// Compute считает итог
func (WeightedPolicy) Compute(lines []FinalLine) float64 { return weightedAverage(lines, false) }

// GradedOnlyPolicy считает взвешенное среднее только по выставленным оценкам.
// Так видно, как студент идёт по уже проверенным работам
type GradedOnlyPolicy struct{}

// Name возвращает название правила
func (GradedOnlyPolicy) Name() string {
	return "только по выставленным оценкам"
}

// Compute считает итог
func (GradedOnlyPolicy) Compute(lines []FinalLine) float64 { return weightedAverage(lines, true) }

// weightedAverage складывает доли от максимума с весами критериев и делит на сумму весов.
// Если skipMissing, критерии без оценки не учитываются совсем
func weightedAverage(lines []FinalLine, skipMissing bool) float64 {
	var sum, weights float64
	for _, l := range lines {
		if l.Evaluation == nil && skipMissing {
			continue
		}
		if l.Evaluation != nil {
			sum += l.Criterion.Weight() * l.Evaluation.Score().Ratio()
		}
		weights += l.Criterion.Weight()
	}
	if weights == 0 {
		return 0
	}
	return MaxFinal * sum / weights
}
