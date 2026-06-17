/*
 ================================================================================
 FILE: phe_indicators_complete.sql
 PURPOSE: Create a materialized view for Public Health Emergency monitoring indicators
 (per CEHS Table 3) from report.hmis_summary with all calculations.
 SOURCE : report.hmis_summary
 GRAIN  : (year, quarter, month, region, district, area, indicator)  -- district level
 NOTES  :
 * Data element IDs were resolved against dhis2_data_elements.csv.
 * Calculated columns now include percentages and composite indicators.
 * Numerator-only counts are returned; percentages derived from numerator/denominator pairs.
 * REFRESH: Use REFRESH MATERIALIZED VIEW CONCURRENTLY report.phe_indicators_district;
 ================================================================================
 */
DROP MATERIALIZED VIEW IF EXISTS report.phe_indicators_district CASCADE;
CREATE MATERIALIZED VIEW report.phe_indicators_district AS WITH indicator_map(
    area,
    indicator,
    data_element_id,
    filter_combo,
    indicator_type
) AS (
    VALUES -- =====================================================================
        -- GENERAL
        -- =====================================================================
        (
            'General',
            'Outpatient attendances or primary care visits',
            'sv6SeKroHPV',
            NULL,
            'count'
        ),
        (
            'General',
            'Outpatient attendances or primary care visits',
            'sQ4EexvvhVe',
            NULL,
            'count'
        ),
        (
            'General',
            'Outpatient attendances or primary care visits',
            'XMObdBBxkoy',
            NULL,
            'count'
        ),
        (
            'General',
            'Outpatient attendances or primary care visits',
            'QtlbL4zTMfF',
            NULL,
            'count'
        ),
        -- =====================================================================
        -- MCH
        -- =====================================================================
        (
            'MCH',
            'Pregnant women with >=1 ANC visit',
            'Q9nSogNmKPt',
            NULL,
            'count'
        ),
        (
            'MCH',
            'Pregnant women with >=1 ANC visit',
            'uUYRrEU5iOB',
            NULL,
            'count'
        ),
        (
            'MCH',
            'Health facility births - live births',
            'fEz9wGsA6YU',
            NULL,
            'count'
        ),
        (
            'MCH',
            'Health facility births - total deliveries',
            'idXOxt69W0e',
            NULL,
            'count'
        ),
        (
            'MCH',
            'Health facility births - total notified',
            'oKKOjhV4TRK',
            NULL,
            'count'
        ),
        (
            'MCH',
            'Health facility births - caesarean sections',
            'sDLD6q8wOCn',
            NULL,
            'count'
        ),
        (
            'MCH',
            'Maternal deaths',
            'F8Iz6QcexWB',
            NULL,
            'count'
        ),
        (
            'MCH',
            'Maternal deaths',
            'JOWj87d62MK',
            NULL,
            'count'
        ),
        (
            'MCH',
            'Neonatal deaths (0-7d)',
            'hrTskGHP0Av',
            NULL,
            'count'
        ),
        (
            'MCH',
            'Neonatal deaths (0-7d)',
            'K1a7iJilOXE',
            NULL,
            'count'
        ),
        (
            'MCH',
            'Neonatal deaths (8-28d)',
            'MBP6dbvM7sj',
            NULL,
            'count'
        ),
        (
            'MCH',
            'Stillbirths - fresh',
            'T8W0wbzErSF',
            NULL,
            'count'
        ),
        (
            'MCH',
            'Stillbirths - fresh',
            'BAGYJJ2kER5',
            NULL,
            'count'
        ),
        (
            'MCH',
            'Stillbirths - fresh (deaths)',
            'cjxTr4s8jLS',
            NULL,
            'count'
        ),
        (
            'MCH',
            'Stillbirths - macerated',
            'ULL9lX3DO7V',
            NULL,
            'count'
        ),
        (
            'MCH',
            'Stillbirths - macerated',
            'DhOt8NQIwPC',
            NULL,
            'count'
        ),
        (
            'MCH',
            'Stillbirths - macerated (deaths)',
            'DhOt8NQIwPC',
            NULL,
            'count'
        ),
        (
            'MCH',
            'Low birth weight (<2.5kg) live births',
            'P1MyPWVxi5T',
            NULL,
            'count'
        ),
        (
            'MCH',
            'Low birth weight (<2.5kg) pre-term births',
            'tVqvqOqsnWI',
            NULL,
            'count'
        ),
        (
            'MCH',
            'Low birth weight (<2.5kg) initiated on KMC',
            'X6rK1GHbLYp',
            NULL,
            'count'
        ),
        (
            'MCH',
            'Immunization - DPT-HepB+Hib 1',
            'Ys31ug5E3f1',
            NULL,
            'count'
        ),
        (
            'MCH',
            'Immunization - DPT-HepB+Hib 2',
            'z1s4aIzf8ga',
            NULL,
            'count'
        ),
        (
            'MCH',
            'Immunization - DPT-HepB+Hib 3',
            'ujs4ipzA4tb',
            NULL,
            'count'
        ),
        (
            'MCH',
            'Immunization - Measles/Rubella 1 (MR1)',
            'WAjgHQVxVVm',
            NULL,
            'count'
        ),
        (
            'MCH',
            'Immunization - Measles/Rubella 2 (MR2)',
            'yctOQDdXZav',
            NULL,
            'count'
        ),
        (
            'MCH',
            'HEI given ARVs at birth (maternity)',
            'OUGMxrtXxri',
            NULL,
            'count'
        ),
        (
            'MCH',
            'HEI initiated ARV prophylaxis 0-6 weeks',
            'Venp30vvLNs',
            NULL,
            'count'
        ),
        (
            'MCH',
            'HEI given ARV prophylaxis at MBCP',
            'D7c8eIfQYNM',
            NULL,
            'count'
        ),
        (
            'MCH',
            'HIV+ pregnant women already on ART before ANC1',
            'mikocWAx5ng',
            NULL,
            'count'
        ),
        (
            'MCH',
            'HIV+ pregnant women on ART - known',
            'C0CnyVY3tm8',
            NULL,
            'count'
        ),
        (
            'MCH',
            'HIV+ pregnant women newly initiated on ART',
            'ZjQgpP9G7m1',
            NULL,
            'count'
        ),
        (
            'MCH',
            'HIV+ pregnant women initiated on ART for eMTCT (any visit)',
            'L4pwIgSDdG6',
            NULL,
            'count'
        ),
        (
            'MCH',
            'Active on ART - pregnant & lactating women (qtr)',
            'WXqcug8MBiR',
            NULL,
            'count'
        ),
        -- =====================================================================
        -- NUTRITION
        -- =====================================================================
        (
            'Nutrition',
            'Mothers initiated breastfeeding within 1hr - total',
            'XXZZbU4B2N3',
            NULL,
            'count'
        ),
        (
            'Nutrition',
            'Mothers initiated breastfeeding within 1hr - HIV+',
            'PNG0YDb6dpy',
            NULL,
            'count'
        ),
        -- =====================================================================
        -- MALARIA (Treatment coverage)
        -- =====================================================================
        (
            'Malaria',
            'Malaria confirmed cases (denominator)',
            'wUDxFVBapIc',
            NULL,
            'denominator'
        ),
        (
            'Malaria',
            'Confirmed malaria cases treated with ACT',
            'VozjcVb9Fu8',
            NULL,
            'numerator'
        ),
        -- =====================================================================
        -- HIV
        -- =====================================================================
        (
            'HIV',
            'Active on ART (total, quarterly)',
            'KybCThzTucw',
            NULL,
            'count'
        ),
        (
            'HIV',
            'Missed appointments (treatment interruption proxy)',
            'mIBmV0slqJC',
            NULL,
            'count'
        ),
        -- =====================================================================
        -- FACILITY DEATHS (HMIS108)
        -- =====================================================================
        (
            'Medical Services',
            'All-cause deaths by ward/service',
            'cufXJq5L6xJ',
            NULL,
            'count'
        ),
        (
            'Medical Services',
            'All-cause deaths (alt form)',
            'vyOajQA5xTu',
            NULL,
            'count'
        ),
        (
            'Medical Services',
            'Total deaths notified',
            'IC6WoTGwSFQ',
            NULL,
            'count'
        ),
        -- =====================================================================
        -- EMHS - Essential Medicine and Health Supplies Stocks
        -- =====================================================================
        (
            'EMHS',
            'ACT - Received',
            'wozPliVW6dd',
            NULL,
            'count'
        ),
        (
            'EMHS',
            'ACT - Dispensed',
            'dsjxSlajJhL',
            NULL,
            'count'
        ),
        (
            'EMHS',
            'Amoxicillin dispersible - Received',
            's5FtDCNBH8y',
            NULL,
            'count'
        ),
        (
            'EMHS',
            'ORS + Zinc - Received',
            'cJm2qpsmczU',
            NULL,
            'count'
        ),
        (
            'EMHS',
            'Gloves - Received',
            'oNuPNEQjtdx',
            NULL,
            'count'
        ),
        (
            'EMHS',
            'RDTs - Received',
            'fmPudbY1m2x',
            NULL,
            'count'
        ),
        (
            'EMHS',
            'Emergency Contraceptives - Received',
            'EyPkT8vx4zY',
            NULL,
            'count'
        ),
        -- Placeholders
        (
            'FP',
            'FP users - oral contraceptives (placeholder)',
            NULL,
            NULL,
            'count'
        ),
        (
            'FP',
            'FP users - injectable contraceptives (placeholder)',
            NULL,
            NULL,
            'count'
        )
),
-- 1) Extract & normalize source rows for indicator DEs
filtered_source AS (
    SELECT m.year,
        m.region,
        m.district,
        m.dataelement,
        m.data_element_id::text AS data_element_id,
        m.category_combo::text AS category_combo,
        COALESCE(m.value::numeric, 0::numeric) AS val,
        -- month: monthly (105-) data only
        CASE
            WHEN m.dataelement ILIKE '105-%'
            AND m.period::text ~ '^[0-9]{6}$' THEN SUBSTRING(
                m.period
                FROM 5 FOR 2
            )::integer
        END AS month,
        -- quarter: from period directly (quarterly) or derived from month (monthly)
        CASE
            WHEN m.period::text ~ '^[0-9]{4}Q[1-4]$' THEN SUBSTRING(
                m.period
                FROM 6 FOR 1
            )::integer
            WHEN m.period::text ~ '^[0-9]{6}$' THEN CASE
                WHEN SUBSTRING(
                    m.period
                    FROM 5 FOR 2
                )::integer BETWEEN 1 AND 3 THEN 1
                WHEN SUBSTRING(
                    m.period
                    FROM 5 FOR 2
                )::integer BETWEEN 4 AND 6 THEN 2
                WHEN SUBSTRING(
                    m.period
                    FROM 5 FOR 2
                )::integer BETWEEN 7 AND 9 THEN 3
                WHEN SUBSTRING(
                    m.period
                    FROM 5 FOR 2
                )::integer BETWEEN 10 AND 12 THEN 4
            END
        END AS quarter
    FROM report.hmis_summary m
    WHERE m.data_element_id::text IN (
            SELECT DISTINCT data_element_id
            FROM indicator_map
            WHERE data_element_id IS NOT NULL
        )
),
-- 2) Join to indicator map (one DE can map to multiple indicators)
tagged AS (
    SELECT fs.year,
        fs.quarter,
        fs.month,
        fs.region,
        fs.district,
        im.area,
        im.indicator,
        im.indicator_type,
        fs.data_element_id,
        fs.dataelement,
        fs.category_combo,
        fs.val
    FROM filtered_source fs
        JOIN indicator_map im ON im.data_element_id = fs.data_element_id
        AND (
            im.filter_combo IS NULL
            OR fs.category_combo ILIKE im.filter_combo
        )
),
-- 3) Aggregate to district level and prepare for calculations
aggregated AS (
    SELECT year,
        quarter,
        month,
        region,
        district,
        area,
        indicator,
        indicator_type,
        SUM(val) AS value
    FROM tagged
    WHERE year IS NOT NULL
        AND quarter IS NOT NULL
    GROUP BY year,
        quarter,
        month,
        region,
        district,
        area,
        indicator,
        indicator_type
),
-- 4) Calculate composite and percentage indicators
calculated AS (
    SELECT year,
        quarter,
        month,
        ('Q' || quarter || ' ' || year) AS qtr_year,
        region,
        district,
        area,
        indicator,
        value,
        indicator_type,
        -- Perinatal/Neonatal/Maternal deaths composite (sum all death types)
        CASE
            WHEN area = 'MCH'
            AND indicator IN (
                'Maternal deaths',
                'Neonatal deaths (0-7d)',
                'Neonatal deaths (8-28d)',
                'Stillbirths - fresh',
                'Stillbirths - macerated',
                'Stillbirths - fresh (deaths)',
                'Stillbirths - macerated (deaths)'
            ) THEN 'Perinatal/Neonatal/Maternal deaths (composite)'
            ELSE NULL
        END AS composite_indicator,
        -- Malaria ACT treatment percentage
        CASE
            WHEN area = 'Malaria'
            AND indicator = 'Confirmed malaria cases treated with ACT' THEN ROUND(
                (
                    value / NULLIF(
                        (
                            SELECT SUM(value)
                            FROM aggregated a2
                            WHERE a2.year = aggregated.year
                                AND a2.quarter = aggregated.quarter
                                AND a2.month IS NOT DISTINCT
                            FROM aggregated.month
                                AND a2.district = aggregated.district
                                AND a2.area = 'Malaria'
                                AND a2.indicator = 'Malaria confirmed cases (denominator)'
                        ),
                        0
                    )
                ) * 100,
                2
            )
            ELSE NULL
        END AS malaria_act_percentage
    FROM aggregated
) -- 5) Final output with all calculated columns
SELECT year,
    quarter,
    month,
    qtr_year,
    region,
    district,
    area,
    indicator,
    ROUND(value::numeric, 2) AS value,
    indicator_type,
    composite_indicator,
    malaria_act_percentage,
    CASE
        WHEN area = 'MCH'
        AND indicator LIKE '%deaths%' THEN 'Mortality'
        WHEN area = 'General' THEN 'Service Utilization'
        WHEN area = 'Malaria' THEN 'Treatment Coverage'
        WHEN area = 'HIV' THEN 'Retention/Continuity'
        WHEN area = 'Nutrition' THEN 'Maternal Health'
        WHEN area = 'Medical Services' THEN 'Hospital Mortality'
        WHEN area = 'EMHS' THEN 'Supply Chain'
        ELSE area
    END AS indicator_category
FROM calculated
WHERE year IS NOT NULL
    AND quarter IS NOT NULL
ORDER BY year DESC,
    quarter DESC,
    month NULLS LAST,
    region,
    district,
    area,
    indicator WITH NO DATA;
-- =====================================================================
-- Indexing for query performance
-- =====================================================================
CREATE INDEX IF NOT EXISTS idx_phe_indicators_district_year_quarter ON report.phe_indicators_district (year DESC, quarter DESC);
CREATE INDEX IF NOT EXISTS idx_phe_indicators_district_geo ON report.phe_indicators_district (region, district, area);
CREATE INDEX IF NOT EXISTS idx_phe_indicators_district_indicator ON report.phe_indicators_district (indicator, indicator_category);
-- =====================================================================
-- Populate the materialized view with data
-- =====================================================================
REFRESH MATERIALIZED VIEW CONCURRENTLY report.phe_indicators_district;
-- =====================================================================
-- REFRESH SCHEDULE:
-- Execute this command periodically (e.g., daily/weekly after data loads):
-- REFRESH MATERIALIZED VIEW CONCURRENTLY report.phe_indicators_district;
-- =====================================================================