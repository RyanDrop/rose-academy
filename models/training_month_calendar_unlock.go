package models

import "time"

// ElapsedCalendarMonths conta meses de calendário entre anchor e ref (inclusivo do mês de anchor como 0).
// Ex.: anchor maio/2026 e ref maio/2026 -> 0; ref junho/2026 -> 1.
// Usa UTC só para ano/mês (dia do mês não altera o índice).
func ElapsedCalendarMonths(anchor, ref time.Time) int {
	a := anchor.UTC()
	r := ref.UTC()
	diff := (r.Year()-a.Year())*12 + int(r.Month()) - int(a.Month())
	if diff < 0 {
		return 0
	}
	return diff
}

// TrainingMonthActiveForPlan indica se o mês de treino monthNumber (1..12) está liberado
// pelo tempo desde a ativação do plano atual (ex.: user_training_selection → selected_at).
// Cada plano (iniciante/intermediário/avançado) tem só 12 meses: ao trocar de plano, a âncora
// reinicia e o mês 1 volta liberado no mês-calendário da nova seleção.
func TrainingMonthActiveForPlan(monthNumber int, planSelectedAt, ref time.Time) bool {
	if monthNumber < 1 || monthNumber > 12 {
		return false
	}
	return ElapsedCalendarMonths(planSelectedAt, ref) >= monthNumber-1
}
