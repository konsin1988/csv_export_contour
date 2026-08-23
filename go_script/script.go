package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

var originColumns = []string{
	"ОрганизацияСсылка", "ОрганизацияНазвание",
	"ОрганизацияИНН", "Год",
	"Период", "Показатель",
	"Значение",
}

var raMetrics = []string{
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
}

var ozMetrics = []string{
	"Cost_Calculator",
	"Post_Sale_Repair_Costs",
}

var rcpMetrics = []string{
	"Contracts_Realize",
	"Cost_Product",
	"Fake_Product_Incidents_Counter",
	"One_Time_Restored_Count",
	"Product_Defect_Fatality_Monitor",
	"Products_Requiring_Restoration",
}

type Row struct {
	CompanyID int64
	Name      string
	Inn       int64
	Year      int64
	Period    string
	Score     string
	Value     float64
	ValueNull bool
}

type Company struct {
	Hash string
	Inn  int64
}

type Result struct {
	Hash      string
	Name      string
	Inn       int64
	Year      int64
	Period    string
	Score     string
	Value     float64
	ValueNull bool
}

type Worker struct {
	db *sql.DB
}

func loadDotEnv(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		os.Setenv(key, value)
	}
	return nil
}

func NewWorker() (*Worker, error) {
	if err := loadDotEnv(".env"); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	user := os.Getenv("DB_USER")
	dbName := os.Getenv("DB_NAME")
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "mysql"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "3306"
	}
	dsn := fmt.Sprintf("%s@tcp(%s:%s)/%s?charset=utf8mb4", user, host, port, dbName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	return &Worker{db: db}, nil
}

func (w *Worker) companyFilepath() string {
	return os.Getenv("BASEDIR") + os.Getenv("COMPANY_FILEPATH")
}

func (w *Worker) resultFilepath() string {
	return os.Getenv("BASEDIR") + os.Getenv("RESULT_FILEPATH")
}

func (w *Worker) readQuery(query string, metrics []string) ([]Row, error) {
	rows, err := w.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	values := make([]sql.NullFloat64, len(metrics))
	scanArgs := make([]interface{}, 5+len(metrics))
	var companyID, year int64
	var name, inn, period string
	scanArgs[0] = &companyID
	scanArgs[1] = &name
	scanArgs[2] = &inn
	scanArgs[3] = &year
	scanArgs[4] = &period
	for i := range values {
		scanArgs[5+i] = &values[i]
	}

	var out []Row
	for rows.Next() {
		if err := rows.Scan(scanArgs...); err != nil {
			return nil, err
		}
		innParsed, err := strconv.ParseInt(strings.TrimSpace(inn), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid inn %q: %w", inn, err)
		}
		base := Row{CompanyID: companyID, Name: name, Inn: innParsed, Year: year, Period: period}
		for i, v := range values {
			row := base
			row.Score = metrics[i]
			if v.Valid {
				row.Value = v.Float64
			} else {
				row.ValueNull = true
			}
			out = append(out, row)
		}
	}
	return out, rows.Err()
}

func readCompanies(path string) ([]Company, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	header, err := r.Read()
	if err != nil {
		return nil, err
	}
	header[0] = strings.TrimPrefix(header[0], "\uFEFF")

	idxHash, idxInn := -1, -1
	for i, col := range header {
		switch col {
		case "Ссылка":
			idxHash = i
		case "ИНН":
			idxInn = i
		}
	}
	if idxHash == -1 {
		return nil, fmt.Errorf("column %q not found in %s", "Ссылка", path)
	}
	if idxInn == -1 {
		return nil, fmt.Errorf("column %q not found in %s", "ИНН", path)
	}

	var records [][]string
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}

	numericHash := true
	for _, rec := range records {
		if _, err := strconv.ParseInt(strings.TrimSpace(rec[idxHash]), 10, 64); err != nil {
			numericHash = false
			break
		}
	}

	out := make([]Company, 0, len(records))
	for _, rec := range records {
		hash := rec[idxHash]
		if numericHash {
			n, err := strconv.ParseInt(strings.TrimSpace(hash), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid Ссылка %q: %w", rec[idxHash], err)
			}
			hash = strconv.FormatInt(n, 10)
		}
		inn, err := strconv.ParseInt(strings.TrimSpace(rec[idxInn]), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid ИНН %q: %w", rec[idxInn], err)
		}
		out = append(out, Company{Hash: hash, Inn: inn})
	}
	return out, nil
}

func mergeInner(companies []Company, concated []Row) []Result {
	byInn := make(map[int64][]Row)
	for _, row := range concated {
		byInn[row.Inn] = append(byInn[row.Inn], row)
	}
	var out []Result
	for _, c := range companies {
		for _, row := range byInn[c.Inn] {
			out = append(out, Result{
				Hash:      c.Hash,
				Name:      row.Name,
				Inn:       row.Inn,
				Year:      row.Year,
				Period:    row.Period,
				Score:     row.Score,
				Value:     row.Value,
				ValueNull: row.ValueNull,
			})
		}
	}
	return out
}

func pyRepr(v float64) string {
	if math.IsNaN(v) {
		return "nan"
	}
	if math.IsInf(v, 1) {
		return "inf"
	}
	if math.IsInf(v, -1) {
		return "-inf"
	}
	if v == 0 {
		if math.Signbit(v) {
			return "-0.0"
		}
		return "0.0"
	}
	e := strconv.FormatFloat(v, 'e', -1, 64)
	exp, err := strconv.Atoi(e[strings.LastIndexAny(e, "eE")+1:])
	if err != nil {
		return e
	}
	if exp >= -4 && exp < 16 {
		s := strconv.FormatFloat(v, 'f', -1, 64)
		if !strings.ContainsAny(s, ".eE") {
			s += ".0"
		}
		return s
	}
	return e
}

func formatValue(v float64, null bool) string {
	if null {
		return ""
	}
	return pyRepr(v)
}

func writeResult(path string, rows []Result) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write(originColumns); err != nil {
		return err
	}
	for _, row := range rows {
		rec := []string{
			row.Hash,
			row.Name,
			strconv.FormatInt(row.Inn, 10),
			strconv.FormatInt(row.Year, 10),
			row.Period,
			row.Score,
			formatValue(row.Value, row.ValueNull),
		}
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	return w.Error()
}

func main() {
	worker, err := NewWorker()
	if err != nil {
		log.Fatal(err)
	}
	defer worker.db.Close()

	dbName := os.Getenv("DB_NAME")
	periodStart := os.Getenv("PERIOD_START")
	periodEnd := os.Getenv("PERIOD_END")
	var whereString string;

	if periodStart != "" && periodEnd != "" {
		whereString = fmt.Sprintf(`
			where period >= '%s'
			and period < '%s'
		`, periodStart, periodEnd)
	} else if periodStart == "" && periodEnd != "" {
		whereString = fmt.Sprintf(`
			where period < '%s'
		`, periodEnd)
	} else if periodStart != "" && periodEnd == "" {
		whereString = fmt.Sprintf(`
			where period >= '%s'
		`, periodStart )
	} else {
		whereString = ""
	}

	queryRA := fmt.Sprintf(`
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
	from %s.companies c 
	join %s.formDataRA ra on c.id = ra.companyId
	%s
	group by c.id, c.name, c.inn, year(ra.period), ra.period
	`, dbName, dbName, whereString)

	queryOZ := fmt.Sprintf(`
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
	from %s.companies c 
	join %s.formDataOZ oz on c.id = oz.companyId 
	%s
	group by c.id, c.name, c.inn, year(oz.period), oz.period 
	`, dbName, dbName, whereString)

	queryRCP := fmt.Sprintf(`
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
	from %s.companies c 
	join %s.formDataRCP rcp on c.id = rcp.companyId  
	%s
	group by c.id, c.name, c.inn, year(rcp.period)
	`, dbName, dbName, whereString)

	ra, err := worker.readQuery(queryRA, raMetrics)
	if err != nil {
		log.Fatal(err)
	}
	oz, err := worker.readQuery(queryOZ, ozMetrics)
	if err != nil {
		log.Fatal(err)
	}
	rcp, err := worker.readQuery(queryRCP, rcpMetrics)
	if err != nil {
		log.Fatal(err)
	}

	concated := append(ra, oz...)
	concated = append(concated, rcp...)

	sort.SliceStable(concated, func(i, j int) bool {
		a, b := concated[i], concated[j]
		if a.CompanyID != b.CompanyID {
			return a.CompanyID < b.CompanyID
		}
		if a.Year != b.Year {
			return a.Year < b.Year
		}
		if a.Period != b.Period {
			return a.Period < b.Period
		}
		return a.Score < b.Score
	})

	companies, err := readCompanies(worker.companyFilepath())
	if err != nil {
		log.Fatal(err)
	}

	result := mergeInner(companies, concated)

	if err := writeResult(worker.resultFilepath(), result); err != nil {
		log.Fatal(err)
	}
}
