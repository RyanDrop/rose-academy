-- Corrige ambiguidade "exercise_name" em get_monthly_progression_summary:
-- em PL/pgSQL, RETURNS TABLE cria variáveis OUT com os mesmos nomes das colunas,
-- o que conflita com exercise_name no CTE sem qualificação.
-- Opcional se a API já pivotar em Go; útil para manter a função no banco alinhada ao script original.

CREATE OR REPLACE FUNCTION get_monthly_progression_summary(
    p_user_id UUID,
    p_month_id UUID
)
RETURNS TABLE(
    exercise_name VARCHAR,
    equipment VARCHAR,
    target_muscle VARCHAR,
    week1_weight DECIMAL,
    week2_weight DECIMAL,
    week3_weight DECIMAL,
    week4_weight DECIMAL,
    initial_weight DECIMAL,
    final_weight DECIMAL,
    weight_gain DECIMAL,
    percentage_gain DECIMAL
) AS $$
BEGIN
    RETURN QUERY
    WITH weekly_data AS (
        SELECT * FROM get_weekly_best_performance(p_user_id, p_month_id)
    ),
    pivoted AS (
        SELECT
            wd.exercise_name AS ename,
            wd.equipment AS eq,
            wd.target_muscle AS tm,
            MAX(CASE WHEN wd.week_number = 1 THEN wd.max_weight END) AS w1,
            MAX(CASE WHEN wd.week_number = 2 THEN wd.max_weight END) AS w2,
            MAX(CASE WHEN wd.week_number = 3 THEN wd.max_weight END) AS w3,
            MAX(CASE WHEN wd.week_number = 4 THEN wd.max_weight END) AS w4
        FROM weekly_data wd
        GROUP BY wd.exercise_name, wd.equipment, wd.target_muscle
    )
    SELECT
        p.ename,
        p.eq,
        p.tm,
        p.w1 AS week1_weight,
        p.w2 AS week2_weight,
        p.w3 AS week3_weight,
        p.w4 AS week4_weight,
        p.w1 AS initial_weight,
        COALESCE(p.w4, p.w3, p.w2, p.w1) AS final_weight,
        COALESCE(p.w4, p.w3, p.w2, p.w1) - p.w1 AS weight_gain,
        CASE
            WHEN p.w1 > 0 THEN
                ROUND(((COALESCE(p.w4, p.w3, p.w2, p.w1) - p.w1) / p.w1 * 100), 2)
            ELSE 0::numeric
        END AS percentage_gain
    FROM pivoted p
    ORDER BY p.ename;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION get_monthly_progression_summary IS 'Resumo de progressão mensal (pivot semanal); CTE qualificada para evitar colunas ambíguas com RETURNS TABLE';
