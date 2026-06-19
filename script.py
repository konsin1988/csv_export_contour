import pandas as pd
from sqlalchemy import create_engine
import pymysql
from dotenv import load_dotenv
import os

load_dotenv()

class Worker:
    def __init__(self):
        self.__set_engine()
        self.origin_columns = [
            'ОрганизацияСсылка', 'ОрганизацияНазвание', 
            'ОрганизацияИНН', 'Год',
           'Период', 'Показатель', 
            'Значение']

    def __load_env(self):
        return [
            os.getenv("DB_USER"),
            os.getenv("DB_PASSWORD"),
            os.getenv("DB_HOST"),
            os.getenv("DB_PORT"),
            os.getenv("DB_NAME"),
               ]

    @property
    def company_filepath(self):
        return os.getenv("BASEDIR") + os.getenv("COMPANY_FILEPATH")

    @property
    def result_filepath(self):
        return os.getenv("BASEDIR") + os.getenv("RESULT_FILEPATH")

    def __set_engine(self):
        user, password, host, port, db_name = self.__load_env()
        password = ":" + password if len(password) > 0 else ""
        self.__engine = create_engine(f"mysql+pymysql://{user}@{host}:{port}/{db_name}")

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
    query_ra = r"""
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
    from asuk_csv_export.companies c 
    join asuk_csv_export.formDataRA ra on c.id = ra.companyId
    group by c.id, c.name, c.inn, year(ra.period), ra.period
    """
    
    query_oz = r"""
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
    from asuk_csv_export.companies c 
    join asuk_csv_export.formDataOZ oz on c.id = oz.companyId 
    group by c.id, c.name, c.inn, year(oz.period), oz.period 
    """
    
    query_rcp = r"""
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
    from asuk_csv_export.companies c 
    join asuk_csv_export.formDataRCP rcp on c.id = rcp.companyId  
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
        .sort_values(['companyId', 'year', 'period', 'score'])
        .assign(inn = lambda x: x['inn'].astype('int64'))
    )

    result = (
        pd.read_csv(worker.company_filepath)
        .rename(columns={
        'Ссылка': 'hash',
        'ИНН': 'inn'
        })
        [['hash', 'inn']]
        .merge(concated, how='inner', on='inn')
        [['hash', 'name', 'inn', 'year', 'period', 'score', 'value']]
    )

    result.columns = worker.origin_columns

    result.to_csv(worker.result_filepath, index=False)

if __name__ == "__main__":
    main()