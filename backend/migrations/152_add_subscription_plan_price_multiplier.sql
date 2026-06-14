ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS price_multiplier DECIMAL(20,8) NOT NULL DEFAULT 1.0;

UPDATE subscription_plans AS p
SET price_multiplier = ROUND(
    (
        p.price /
        NULLIF(
            (
                CASE
                    WHEN COALESCE(NULLIF(p.validity_unit, ''), 'days') IN ('week', 'weeks')
                        THEN g.weekly_limit_usd
                    WHEN COALESCE(NULLIF(p.validity_unit, ''), 'days') IN ('month', 'months')
                        THEN g.monthly_limit_usd
                    ELSE g.daily_limit_usd
                END
                / NULLIF(g.rate_multiplier, 0)
            )
            * NULLIF(p.validity_days, 0),
            0
        )
    )::numeric,
    8
)
FROM groups AS g
WHERE p.group_id = g.id
  AND p.price > 0
  AND p.validity_days > 0
  AND g.rate_multiplier > 0
  AND (
    CASE
        WHEN COALESCE(NULLIF(p.validity_unit, ''), 'days') IN ('week', 'weeks')
            THEN g.weekly_limit_usd
        WHEN COALESCE(NULLIF(p.validity_unit, ''), 'days') IN ('month', 'months')
            THEN g.monthly_limit_usd
        ELSE g.daily_limit_usd
    END
  ) > 0;
