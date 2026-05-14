-- =============================================================================
-- APAGA progresso de treino + weekly_workouts / workout_exercises.
-- Destrutivo: apenas desenvolvimento. Faca backup se precisar dos dados.
--
-- Nao apaga public.exercises (biblioteca), training_months, planos nem
-- user_training_selection.
--
-- psql "$DATABASE_URL" -f db/wipe_workouts_and_training_progress.sql
-- Depois: go run ./cmd/seedworkouts
-- =============================================================================

BEGIN;

-- Tabelas de series (nome depende do schema das funcoes SQL no PG).
-- Lista extensivel; TRUNCATE so corre se a tabela existir.
DO $$
DECLARE
    t text;
BEGIN
    FOREACH t IN ARRAY ARRAY[
        'user_exercise_sets',
        'user_workout_exercise_sets',
        'user_workout_sets',
        'workout_set_logs',
        'user_workout_set_logs',
        'exercise_set_logs'
    ]
    LOOP
        IF to_regclass('public.' || t) IS NOT NULL THEN
            EXECUTE format('TRUNCATE TABLE public.%I RESTART IDENTITY CASCADE', t);
        END IF;
    END LOOP;
END $$;

DELETE FROM user_workout_completion;
DELETE FROM user_month_progress;
DELETE FROM workout_exercises;
DELETE FROM weekly_workouts;

COMMIT;
