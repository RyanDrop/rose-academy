-- Migra views de subscriptions -> user_subscriptions + products (Rose Academy / Abacate).
-- Backup antes de executar. Executar o ficheiro inteiro (todas as instrucoes).

DROP VIEW IF EXISTS public.active_trials;
DROP VIEW IF EXISTS public.active_users;

CREATE VIEW public.active_users AS
SELECT
    sub.id,
    sub.email,
    sub.full_name,
    sub.gender,
    sub.goal_name,
    sub.subscription_status,
    sub.is_trial,
    sub.trial_end_date,
    sub.subscription_end_date,
    sub.plan_type,
    sub.access_type,
    sub.days_remaining,
    sub.user_since
FROM (
    SELECT DISTINCT ON (u.id)
        u.id,
        u.email,
        u.full_name,
        u.gender,
        g.name AS goal_name,
        us.status::text AS subscription_status,
        false AS is_trial,
        NULL::timestamptz AS trial_end_date,
        NULL::timestamptz AS subscription_end_date,
        COALESCE(p.name::text, '') AS plan_type,
        'Assinatura Paga'::text AS access_type,
        NULL::integer AS days_remaining,
        u.created_at AS user_since
    FROM users u
    INNER JOIN user_subscriptions us
        ON us.user_id = u.id AND us.status = 'PAID'
    LEFT JOIN products p ON p.id = us.product_id
    LEFT JOIN goals g ON g.id = u.goal_id
    WHERE u.is_active = true
    ORDER BY u.id, us.paid_at DESC NULLS LAST, us.updated_at DESC
) sub;

CREATE VIEW public.active_trials AS
SELECT *
FROM public.active_users
WHERE is_trial = true;

-- Apos validar:
-- DROP TABLE IF EXISTS public.subscriptions CASCADE;
-- DROP TYPE IF EXISTS public.subscription_status_type CASCADE;
