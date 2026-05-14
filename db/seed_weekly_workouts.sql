-- =============================================================================
-- Seed: 5 treinos por mês (dia 1-5 = segunda a sexta), 6 exercicios por dia.
-- Cada dia da semana tem um bloco diferente (offset (dia-1)*6 na lista ordenada
-- por nome, com wrap pelo total de exercicios). O mesmo par (mes, day_number)
-- representa aquele dia em todas as semanas do mês (~4 semanas).
--
-- Reexecutavel: se weekly_workouts ja existir para (mes, dia), APAGA e recria
-- apenas os workout_exercises desse treino e reinsere o circuito (para nao
-- ficar preso ao seed antigo com os mesmos 6 exercicios em todos os dias).
-- Se DELETE em workout_exercises falhar por FK (series ainda ligadas), rode
-- antes: psql ... -f db/wipe_workouts_and_training_progress.sql
--
-- Pre-requisito: training_months + exercises ativos (scripts partes 4 e 5).
--
--   go run ./cmd/seedworkouts
--   psql "$DATABASE_URL" -f db/seed_weekly_workouts.sql
-- =============================================================================

DO $$
DECLARE
    tm         RECORD;
    d          INT;
    wid        UUID;
    ex_count   INT;
    ins_ww     INT := 0;
    sync_ex    INT := 0;
    day_titles TEXT[] := ARRAY[
        'Circuito segunda - Ativacao e base',
        'Circuito terca - Pernas e gluteos',
        'Circuito quarta - Costas e posterior',
        'Circuito quinta - Peito, ombros e bracos',
        'Circuito sexta - Full body e core'
    ];
BEGIN
    SELECT COUNT(*) INTO ex_count FROM exercises WHERE is_active = true;
    IF ex_count = 0 THEN
        RAISE EXCEPTION 'Nenhum exercicio ativo em exercises. Insira a biblioteca (parte 5) antes.';
    END IF;

    FOR tm IN
        SELECT id, month_number, plan_id
        FROM training_months
        WHERE is_active = true
        ORDER BY plan_id, month_number
    LOOP
        FOR d IN 1..5 LOOP
            wid := NULL;
            SELECT ww.id INTO wid
            FROM weekly_workouts ww
            WHERE ww.month_id = tm.id AND ww.day_number = d AND ww.is_active = true
            LIMIT 1;

            IF wid IS NULL THEN
                INSERT INTO weekly_workouts (
                    month_id,
                    day_number,
                    title,
                    description,
                    order_position,
                    is_active
                )
                VALUES (
                    tm.id,
                    d,
                    day_titles[d],
                    'Circuito do dia: execute em sequencia; respeite descanso entre estacoes.',
                    d,
                    true
                )
                RETURNING id INTO wid;
                ins_ww := ins_ww + 1;
            ELSE
                DELETE FROM workout_exercises WHERE workout_id = wid;
                sync_ex := sync_ex + 1;
            END IF;

            INSERT INTO workout_exercises (
                workout_id,
                exercise_id,
                order_position,
                sets,
                reps,
                rest_time,
                is_active
            )
            SELECT
                wid,
                r.id,
                gs.slot::int,
                3,
                '10-12',
                50,
                true
            FROM generate_series(1, 6) AS gs(slot)
            INNER JOIN (
                SELECT e.id, row_number() OVER (ORDER BY e.name) AS rn
                FROM exercises e
                WHERE e.is_active = true
            ) r ON r.rn = (
                ((d - 1) * 6 + gs.slot::int - 1) % ex_count
            ) + 1;

        END LOOP;
    END LOOP;

    RAISE NOTICE 'weekly_workouts novos: %; treinos existentes com exercicios atualizados: %', ins_ww, sync_ex;
END $$;
