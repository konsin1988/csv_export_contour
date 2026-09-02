package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	_ "sort"
	"strconv"
	"strings"
	"time"

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

type CompanyInfo struct {
	Hash string
	Name string
	Inn int64
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

type YearPeriod struct {
	Year   int64
	Period string
}

type Worker struct {
	db *sql.DB
}

func loadDotEnv(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for line := range strings.SplitSeq(string(data), "\n") {
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
	scanArgs := make([]any, 5+len(metrics))
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

func getPeriodRange() []YearPeriod {
	startStr := os.Getenv("PERIOD_START")
	endStr := os.Getenv("PERIOD_END")

	// Defaults
	start := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Now()

	var err error

	if startStr != "" {
		start, err = time.Parse("2006-01-02", startStr)
		if err != nil {
			log.Fatalf("invalid PERIOD_START: %v", err)
		}
	}

	if endStr != "" {
		end, err = time.Parse("2006-01-02", endStr)
		if err != nil {
			log.Fatalf("invalid PERIOD_END: %v", err)
		}
	}

	startYear, startQuarter := start.Year(), (int(start.Month())-1)/3+1
	endYear, endQuarter := end.Year(), (int(end.Month())-1)/3+1

	var result []YearPeriod

	for year := startYear; year <= endYear; year++ {
		firstQuarter := 1
		lastQuarter := 4

		if year == startYear {
			firstQuarter = startQuarter
		}

		if year == endYear {
			lastQuarter = endQuarter
		}

		for quarter := firstQuarter; quarter <= lastQuarter; quarter++ {
			result = append(result, YearPeriod{
				Year:   int64(year),
				Period: fmt.Sprintf("Q%d", quarter),
			})
		}
	}

	return result
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

func mergeCompanies(companies []Company, concated []Row) []CompanyInfo {
	byInn := make(map[int64]Company, len(companies))

	for _, company := range companies {
		byInn[company.Inn] = company
	}

	// Hash -> ConcatedCompany.
	// Map guarantees that Hash is unique.
	byHash := make(map[string]CompanyInfo)

	for _, row := range concated {
		company, exists := byInn[row.Inn]
		if !exists {
			continue
		}

		if _, exists := byHash[company.Hash]; exists {
			continue
		}

		byHash[company.Hash] = CompanyInfo{
			Hash: company.Hash,
			Name: row.Name,
			Inn:  row.Inn,
		}
	}

	result := make([]CompanyInfo, 0, len(byHash))

	for _, company := range byHash {
		result = append(result, company)
	}
	return result
}

func mergeInner(
	companies []CompanyInfo,
	concated []Row,
	periods []YearPeriod,
	metrics []string,
) []Result {

	type key struct {
		hash   string
		year   int64
		period string
		score  string
	}

	// Existing metric data.
	byKey := make(map[key]Row, len(concated))

	// We need to know which Hash belongs to which Row.
	// Build Inn -> Hash from ConcatedCompany.
	hashByInn := make(map[int64]string, len(companies))

	for _, company := range companies {
		hashByInn[company.Inn] = company.Hash
	}

	for _, row := range concated {
		hash, exists := hashByInn[row.Inn]
		if !exists {
			continue
		}

		k := key{
			hash:   hash,
			year:   row.Year,
			period: row.Period,
			score:  row.Score,
		}

		byKey[k] = row
	}

	var out []Result

	for _, company := range companies {
		for _, p := range periods {
			for _, metric := range metrics {

				k := key{
					hash:   company.Hash,
					year:   p.Year,
					period: p.Period,
					score:  metric,
				}

				row, exists := byKey[k]

				if exists {
					// Existing data.
					out = append(out, Result{
						Hash:      company.Hash,
						Name:      company.Name,
						Inn:       company.Inn,
						Year:      row.Year,
						Period:    row.Period,
						Score:     row.Score,
						Value:     row.Value,
						ValueNull: row.ValueNull,
					})
				} else {
					// Missing period/metric.
					out = append(out, Result{
						Hash:      company.Hash,
						Name:      company.Name,
						Inn:       company.Inn,
						Year:      p.Year,
						Period:    p.Period,
						Score:     metric,
						ValueNull: true,
					})
				}
			}
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
	return strconv.FormatFloat(v, 'f', -1, 64)
	//return pyRepr(v)
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

	companies, err := readCompanies(worker.companyFilepath())
	if err != nil {
		log.Fatal(err)
	}

	periods := getPeriodRange()

	metrics := append([]string{}, raMetrics...)
	metrics = append(metrics, ozMetrics...)
	metrics = append(metrics, rcpMetrics...)

	mergedCompanies := mergeCompanies(companies, concated)

	result := mergeInner(mergedCompanies, concated, periods, metrics)

	//sort.SliceStable(result, func(i, j int) bool {
	//	a, b := result[i], result[j]
	//	if a.CompanyID != b.CompanyID {
	//		return a.CompanyID < b.CompanyID
	//	}
	//	if a.Year != b.Year {
	//		return a.Year < b.Year
	//	}
	//	if a.Period != b.Period {
	//		return a.Period < b.Period
	//	}
	//	return a.Score < b.Score
	//})

	if err := writeResult(worker.resultFilepath(), result); err != nil {
		log.Fatal(err)
	}
}
