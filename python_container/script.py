import pandas as pd
from sqlalchemy import create_engine
import pymysql
from dotenv import load_dotenv
import os
from datetime import datetime

load_dotenv()

class Worker:
    def __init__(self):
        self.__set_engine()
        self.origin_columns = [
            'ОрганизацияСсылка', 'ОрганизацияНазвание', 
            'ОрганизацияИНН', 'Год',
           'Период', 'Показатель', 
            'Значение']

        self.__raMetrics = [ 
        	"Accepted_Claims",
        	"Achieved_Reliability_Metrics",
        	"Claims_count",
        	"Construction_Defects_Total",
        	"Contract_Quality_Failure_Tracker",
        	"Deviation_Approvals_Count",
        	"Factory_Defects_Total",
        	"Factory_PKI_Defect_Total",
        	"False_Defects_Total",
        	"First_Pass_Accepted_Count",
        	"Manufactured_Items_Total",
        	"Mean_Time_To_Restoration",
        	"Other_Defect_Total",
        	"Presented_Product_Count",
        	"Suspension_Tracker",
        	"Target_Reliability_Metrics",
        	"Track_Defects",
        	"track_Factory_Defects",
        	"track_Usage_Defects",
        	"Unknown_Defects_Total",
        	"Usage_Defects_Total",
        	"Usage_PKI_Defect_Total",
        	"Warranty_Product",
                            ]
        
        self.__ozMetrics = [
        	"Cost_Calculator",
        	"Post_Sale_Repair_Costs",
        ]
        
        self.__rcpMetrics = [
        	"Contracts_Realize",
        	"Cost_Product",
        	"Fake_Product_Incidents_Counter",
        	"One_Time_Restored_Count",
        	"Product_Defect_Fatality_Monitor",
        	"Products_Requiring_Restoration",
        ]

    def __load_db_env(self):
        return [
            os.getenv("DB_USER"),
            os.getenv("DB_PASSWORD"),
            os.getenv("DB_HOST"),
            os.getenv("DB_PORT"),
            os.getenv("DB_NAME"),
               ]

    def __load_date_range_env(self):
        period_start = os.getenv("PERIOD_START")
        period_end = os.getenv("PERIOD_END")
        now_dt = datetime.now()
        first_month_of_quarter = 3 * ((now_dt.month - 1) // 3) 
        period_now = datetime(now_dt.year, first_month_of_quarter + 3, 1).strftime("%Y-%m-%d")
        period_first = '2019-03-01'

        if period_start != "" and period_end != "": 
	        return [period_start, period_end]
        elif period_start == "" and period_end != "":
	        return [period_first, period_end]
        elif period_start != "" and period_end == "":
	        return [period_start, period_now]
        else:
            return [period_first, period_now]

    def __get_periods(self):
        first_period, last_period = self.__load_date_range_env()
        start_date = pd.to_datetime(first_period)
        end_date = pd.to_datetime(last_period)
        
        years = []
        periods = []
        
        for year in range(start_date.year, end_date.year + 1):
            period_start = 1
            period_end = 4 
            if year == start_date.year:
                period_start = (start_date.month - 1)//3+1 
            if year == end_date.year:
                period_end = (end_date.month - 1)//3+1
            for period in range(period_start, period_end+1):
                years += [year]
                periods += [f"Q{period}"]
        
        return pd.DataFrame({"year": years, "period": periods})

    @property
    def periods(self)->pd.DataFrame:
        periods = self.__get_periods()
        return periods

    @property
    def where_string(self):
        period_start, period_end = self.__load_date_range_env()
        res = f"where period >= '{period_start}' and period < '{period_end}'"
        return res 

    @property
    def metrics(self):
        return pd.DataFrame({"score": self.__raMetrics + self.__ozMetrics + self.__rcpMetrics}) 

    @property
    def company_filepath(self):
        return os.getenv("BASEDIR") + os.getenv("COMPANY_FILEPATH")

    @property
    def result_filepath(self):
        return os.getenv("BASEDIR") + os.getenv("RESULT_FILEPATH")

    @property
    def db_name(self):
        user, password, host, port, db_name = self.__load_db_env()
        return db_name 

    def __set_engine(self):
        user, password, host, port, db_name = self.__load_db_env()
        password = ":" + password if len(password) > 0 else ""
        self.__engine = create_engine(f"mysql+pymysql://{user}{password}@{host}:{port}/{db_name}")

    def read_query(self, query):
        with self.__engine.connect() as conn:
            df = pd.read_sql_query(query, conn)
        return df
    
    def execute_sql(self, query):
        with self.__engine.connect() as conn:
            conn.execute(query)

    def reshape(self, df: pd.DataFrame):
        return pd.melt(
            df,
            id_vars=['companyId', 'name', 'inn', 'year', 'period'],
            value_vars=[col for col in df.columns if col not in 
                        ['companyId', 'name', 'inn', 'year', 'period']],
            var_name='score',
            value_name='value'
        )




def main():
    worker = Worker()
    db_name = worker.db_name
    where_str = worker.where_string
    query_ra = fr"""
    select
    	c.id as companyId, c.name, c.inn,
    	year(ra.period) as year,
    	case
    		when month(ra.period) = 3 then 'Q1'
    		when month(ra.period) = 6 then 'Q2'
    		when month(ra.period) = 9 then 'Q3'
    		when month(ra.period) = 12 then 'Q4' end as period,
    	sum(ra.field14) as Accepted_Claims,
    	sum(ra.field11) as Achieved_Reliability_Metrics,
    	sum(ra.field13) as Claims_count,
    	sum(ra.field18) as Construction_Defects_Total,
    	sum(ra.field9) as Contract_Quality_Failure_Tracker,
    	sum(ra.field9) as Deviation_Approvals_Count,
    	sum(ra.field19) as Factory_Defects_Total,
    	sum(ra.field20) as Factory_PKI_Defect_Total,
    	sum(ra.field341) as False_Defects_Total,
    	sum(ra.field7) as First_Pass_Accepted_Count,
    	sum(ra.field8) as Manufactured_Items_Total,
    	sum(ra.field34) as Mean_Time_To_Restoration,
    	sum(ra.field24) as Other_Defect_Total,
    	sum(ra.field6) as Presented_Product_Count,
    	sum(ra.field10) as Suspension_Tracker,
    	sum(ra.field12) as Target_Reliability_Metrics,
    	sum(ra.field5) as Track_Defects,
    	sum(ra.field161) as track_Factory_Defects,
    	sum(ra.field162) as track_Usage_Defects,
    	sum(ra.field23) as Unknown_Defects_Total,
    	sum(ra.field22) as Usage_Defects_Total,
    	sum(ra.field21) as Usage_PKI_Defect_Total,
    	sum(ra.field2) as Warranty_Product
    from {db_name}.companies c 
    join {db_name}.formDataRA ra on c.id = ra.companyId
    {where_str}
    group by c.id, c.name, c.inn, year(ra.period), ra.period
    """
    
    query_oz = fr"""
    select 
    	c.id as companyId, c.name, c.inn,
    	year(oz.period) as year,
    	case
    		when month(oz.period) = 3 then 'Q1'
    		when month(oz.period) = 6 then 'Q2'
    		when month(oz.period) = 9 then 'Q3'
    		when month(oz.period) = 12 then 'Q4' end as period,
    	sum(oz.field13) as Cost_Calculator,
    	sum(oz.field7) as Post_Sale_Repair_Costs
    from {db_name}.companies c 
    join {db_name}.formDataOZ oz on c.id = oz.companyId 
    {where_str}
    group by c.id, c.name, c.inn, year(oz.period), oz.period 
    """
    
    query_rcp = fr"""
    select
    	c.id as companyId, c.name, c.inn,
    	year(rcp.period) as year,
    	'Q4' as period,
    	sum(rcp.field10) as Contracts_Realize,
    	sum(rcp.field2) as Cost_Product,
    	sum(rcp.field131) as Fake_Product_Incidents_Counter,
    	sum(rcp.field8) as One_Time_Restored_Count,
    	sum(rcp.field1) as Product_Defect_Fatality_Monitor,
    	sum(rcp.field7) as Products_Requiring_Restoration
    from {db_name}.companies c 
    join {db_name}.formDataRCP rcp on c.id = rcp.companyId  
    {where_str}
    group by c.id, c.name, c.inn, year(rcp.period)
    """

    ra_df = worker.read_query(query_ra)
    oz_df = worker.read_query(query_oz)
    rcp_df = worker.read_query(query_rcp)

    concated = (
        pd.concat([
            worker.reshape(ra_df), 
            worker.reshape(oz_df), 
            worker.reshape(rcp_df)
        ])
        .assign(inn = lambda x: x['inn'].astype('int64'))
    )

    companies = (
        pd.read_csv(worker.company_filepath)
        .rename(columns={
        'Ссылка': 'hash',
        'ИНН': 'inn'
        })
        .dropna(subset=['inn'])
        [['hash', 'inn']]
    )

    companies_with_names = (
        concated
        .merge(companies, on='inn', how='inner')
        [['hash', 'name', 'inn']]
        .drop_duplicates(keep='first')
    )

    metrics = worker.metrics
    periods = worker.periods


    result = (
        periods
        .merge(metrics, how='cross')
        .merge(companies_with_names, how='cross')
        .merge(concated, on=['name', 'inn', 'year', 'period', 'score'], how='left')
        .assign(value=lambda x: x['value'].fillna(0))
        [['hash', 'name', 'inn', 'year', 'period', 'score', 'value']]
    )

    result.columns = worker.origin_columns

    result.to_csv(worker.result_filepath, index=False)

if __name__ == "__main__":
    main()
