WITH ra_agg AS (
    SELECT
        c.id AS companyId,
        c.name,
        c.inn,
        YEAR(ra.period) AS year,
        CASE
            WHEN MONTH(ra.period) = 3 THEN 'Q1'
            WHEN MONTH(ra.period) = 6 THEN 'Q2'
            WHEN MONTH(ra.period) = 9 THEN 'Q3'
            WHEN MONTH(ra.period) = 12 THEN 'Q4'
        END AS period,
        SUM(ra.field14) AS Accepted_Claims,
        SUM(ra.field11) AS Achieved_Reliability_Metrics,
        SUM(ra.field13) AS Claims_count,
        SUM(ra.field18) AS Construction_Defects_Total,
        SUM(ra.field9)  AS Contract_Quality_Failure_Tracker,
        SUM(ra.field9)  AS Deviation_Approvals_Count,
        SUM(ra.field19) AS Factory_Defects_Total,
        SUM(ra.field20) AS Factory_PKI_Defect_Total,
        SUM(ra.field341) AS False_Defects_Total,
        SUM(ra.field7)  AS First_Pass_Accepted_Count,
        SUM(ra.field8)  AS Manufactured_Items_Total,
        SUM(ra.field34) AS Mean_Time_To_Restoration,
        SUM(ra.field24) AS Other_Defect_Total,
        SUM(ra.field6)  AS Presented_Product_Count,
        SUM(ra.field10) AS Suspension_Tracker,
        SUM(ra.field12) AS Target_Reliability_Metrics,
        SUM(ra.field5)  AS Track_Defects,
        SUM(ra.field161) AS track_Factory_Defects,
        SUM(ra.field162) AS track_Usage_Defects,
        SUM(ra.field23) AS Unknown_Defects_Total,
        SUM(ra.field22) AS Usage_Defects_Total,
        SUM(ra.field21) AS Usage_PKI_Defect_Total,
        SUM(ra.field2)  AS Warranty_Product
    FROM `${DB_NAME}`.companies c
    JOIN `${DB_NAME}`.formDataRA ra ON c.id = ra.companyId
    GROUP BY c.id, c.name, c.inn, YEAR(ra.period), ra.period
),
oz_agg AS (
    SELECT
        c.id AS companyId,
        c.name,
        c.inn,
        YEAR(oz.period) AS year,
        CASE
            WHEN MONTH(oz.period) = 3 THEN 'Q1'
            WHEN MONTH(oz.period) = 6 THEN 'Q2'
            WHEN MONTH(oz.period) = 9 THEN 'Q3'
            WHEN MONTH(oz.period) = 12 THEN 'Q4'
        END AS period,
        SUM(oz.field13) AS Cost_Calculator,
        SUM(oz.field7)  AS Post_Sale_Repair_Costs
    FROM `${DB_NAME}`.companies c
    JOIN `${DB_NAME}`.formDataOZ oz ON c.id = oz.companyId
    GROUP BY c.id, c.name, c.inn, YEAR(oz.period), oz.period
),
rcp_agg AS (
    SELECT
        c.id AS companyId,
        c.name,
        c.inn,
        YEAR(rcp.period) AS year,
        'Q4' AS period,
        SUM(rcp.field10) AS Contracts_Realize,
        SUM(rcp.field2)  AS Cost_Product,
        SUM(rcp.field131) AS Fake_Product_Incidents_Counter,
        SUM(rcp.field8)  AS One_Time_Restored_Count,
        SUM(rcp.field1)  AS Product_Defect_Fatality_Monitor,
        SUM(rcp.field7)  AS Products_Requiring_Restoration
    FROM `${DB_NAME}`.companies c
    JOIN `${DB_NAME}`.formDataRCP rcp ON c.id = rcp.companyId
    GROUP BY c.id, c.name, c.inn, YEAR(rcp.period)
),
melted_ra AS (
    SELECT companyId, name, inn, year, period, 'Accepted_Claims' AS score, Accepted_Claims AS value FROM ra_agg UNION ALL
    SELECT companyId, name, inn, year, period, 'Achieved_Reliability_Metrics', Achieved_Reliability_Metrics FROM ra_agg UNION ALL
    SELECT companyId, name, inn, year, period, 'Claims_count', Claims_count FROM ra_agg UNION ALL
    SELECT companyId, name, inn, year, period, 'Construction_Defects_Total', Construction_Defects_Total FROM ra_agg UNION ALL
    SELECT companyId, name, inn, year, period, 'Contract_Quality_Failure_Tracker', Contract_Quality_Failure_Tracker FROM ra_agg UNION ALL
    SELECT companyId, name, inn, year, period, 'Deviation_Approvals_Count', Deviation_Approvals_Count FROM ra_agg UNION ALL
    SELECT companyId, name, inn, year, period, 'Factory_Defects_Total', Factory_Defects_Total FROM ra_agg UNION ALL
    SELECT companyId, name, inn, year, period, 'Factory_PKI_Defect_Total', Factory_PKI_Defect_Total FROM ra_agg UNION ALL
    SELECT companyId, name, inn, year, period, 'False_Defects_Total', False_Defects_Total FROM ra_agg UNION ALL
    SELECT companyId, name, inn, year, period, 'First_Pass_Accepted_Count', First_Pass_Accepted_Count FROM ra_agg UNION ALL
    SELECT companyId, name, inn, year, period, 'Manufactured_Items_Total', Manufactured_Items_Total FROM ra_agg UNION ALL
    SELECT companyId, name, inn, year, period, 'Mean_Time_To_Restoration', Mean_Time_To_Restoration FROM ra_agg UNION ALL
    SELECT companyId, name, inn, year, period, 'Other_Defect_Total', Other_Defect_Total FROM ra_agg UNION ALL
    SELECT companyId, name, inn, year, period, 'Presented_Product_Count', Presented_Product_Count FROM ra_agg UNION ALL
    SELECT companyId, name, inn, year, period, 'Suspension_Tracker', Suspension_Tracker FROM ra_agg UNION ALL
    SELECT companyId, name, inn, year, period, 'Target_Reliability_Metrics', Target_Reliability_Metrics FROM ra_agg UNION ALL
    SELECT companyId, name, inn, year, period, 'Track_Defects', Track_Defects FROM ra_agg UNION ALL
    SELECT companyId, name, inn, year, period, 'track_Factory_Defects', track_Factory_Defects FROM ra_agg UNION ALL
    SELECT companyId, name, inn, year, period, 'track_Usage_Defects', track_Usage_Defects FROM ra_agg UNION ALL
    SELECT companyId, name, inn, year, period, 'Unknown_Defects_Total', Unknown_Defects_Total FROM ra_agg UNION ALL
    SELECT companyId, name, inn, year, period, 'Usage_Defects_Total', Usage_Defects_Total FROM ra_agg UNION ALL
    SELECT companyId, name, inn, year, period, 'Usage_PKI_Defect_Total', Usage_PKI_Defect_Total FROM ra_agg UNION ALL
    SELECT companyId, name, inn, year, period, 'Warranty_Product', Warranty_Product FROM ra_agg
),
melted_oz AS (
    SELECT companyId, name, inn, year, period, 'Cost_Calculator' AS score, Cost_Calculator AS value FROM oz_agg UNION ALL
    SELECT companyId, name, inn, year, period, 'Post_Sale_Repair_Costs', Post_Sale_Repair_Costs FROM oz_agg
),
melted_rcp AS (
    SELECT companyId, name, inn, year, period, 'Contracts_Realize' AS score, Contracts_Realize AS value FROM rcp_agg UNION ALL
    SELECT companyId, name, inn, year, period, 'Cost_Product', Cost_Product FROM rcp_agg UNION ALL
    SELECT companyId, name, inn, year, period, 'Fake_Product_Incidents_Counter', Fake_Product_Incidents_Counter FROM rcp_agg UNION ALL
    SELECT companyId, name, inn, year, period, 'One_Time_Restored_Count', One_Time_Restored_Count FROM rcp_agg UNION ALL
    SELECT companyId, name, inn, year, period, 'Product_Defect_Fatality_Monitor', Product_Defect_Fatality_Monitor FROM rcp_agg UNION ALL
    SELECT companyId, name, inn, year, period, 'Products_Requiring_Restoration', Products_Requiring_Restoration FROM rcp_agg
),
melted_all AS (
    SELECT * FROM melted_ra UNION ALL
    SELECT * FROM melted_oz UNION ALL
    SELECT * FROM melted_rcp
),
melted_sorted AS (
    SELECT
        companyId,
        name,
        CAST(inn AS SIGNED) AS inn,
        year,
        period,
        score,
        value
    FROM melted_all
)
SELECT
    cf.`hash` AS `ОрганизацияСсылка`,
    m.name    AS `ОрганизацияНазвание`,
    m.inn     AS `ОрганизацияИНН`,
    m.year    AS `Год`,
    m.period  AS `Период`,
    m.score   AS `Показатель`,
    m.value   AS `Значение`
FROM melted_sorted m
JOIN company_file cf ON cf.inn = m.inn
ORDER BY m.companyId, m.year, m.period, m.score;

