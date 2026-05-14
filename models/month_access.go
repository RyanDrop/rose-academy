package models

import "github.com/google/uuid"

// MonthAccessChecker verifica se um mês de treino pertence ao plano ativo do usuário.
type MonthAccessChecker interface {
	IsMonthInUserActivePlan(userID, monthID uuid.UUID) (bool, error)
}
