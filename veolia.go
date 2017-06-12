package veolia

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/extrame/xls"
	"github.com/n0rad/go-erlog/data"
	"github.com/n0rad/go-erlog/errs"
)

type StatementType rune

const (
	Measured  StatementType = 'M'
	Estimated StatementType = 'E'
)

type DailyConsumption struct {
	Day         time.Time
	Index       int
	Consumption int
	Type        StatementType
}

type Veolia struct {
	Host     string
	Username string
	Password string

	client *http.Client
}

func NewVeolia() *Veolia {
	return &Veolia{
		Host: "https://www.service-client.veoliaeau.fr",
	}
}

func (v *Veolia) getConsumption() ([]DailyConsumption, error) {
	cookieJar, _ := cookiejar.New(nil)
	client := http.Client{
		Jar: cookieJar,
	}

	getFunc := func(fullUrl string) (*http.Response, error) {
		return client.Get(fullUrl)
	}

	content, err := v.callURL(func(fullUrl string) (*http.Response, error) {
		return client.PostForm(fullUrl, url.Values{
			"veolia_username": {v.Username},
			"veolia_password": {v.Password},
			"login":           {"OK"},
		})
	}, "/home.loginAction.do")
	if err != nil {
		return []DailyConsumption{}, errs.WithEF(err, data.WithField("username", v.Username), "Failed to login")
	}
	if strings.Contains(string(content), "/home/connexion-espace-client.loginAction.do") {
		return []DailyConsumption{}, errs.WithEF(err, data.WithField("username", v.Username), "Login failed, login form is still there")
	}

	content, err = v.callURL(getFunc, "/home/espace-client/votre-consommation.html?vueConso=historique") // mandatory call
	if err != nil || strings.Contains(string(content), "momentanément indisponible") {
		return []DailyConsumption{}, errs.WithE(err, "Veolia consumption is temporary unavailable")
	}
	report, err := v.callURL(getFunc, "/home/espace-client/votre-consommation.exportConsommationData.do?vueConso=historique")
	if err != nil {
		return []DailyConsumption{}, err
	}
	return readConsumptionXls(report)
}

func (v *Veolia) callURL(f func(fullUrl string) (*http.Response, error), path string) ([]byte, error) {
	resp, err := f(v.Host + path)
	if err != nil {
		return []byte{}, errs.WithEF(err, data.WithField("url", v.Host+path), "Url call failed")
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return b, errs.WithEF(err, data.WithField("url", v.Host+path), "Failed to read call response")
	}
	return b, nil
}

func readConsumptionXls(b []byte) ([]DailyConsumption, error) {
	conso := []DailyConsumption{}
	xlFile, err := xls.OpenReader(bytes.NewReader(b), "utf-8")
	if err != nil {
		return conso, errs.WithE(err, "Failed to open consumption xls file")
	}

	sheet1 := xlFile.GetSheet(0)
	if sheet1 == nil {
		return conso, errs.WithE(err, "Empty xls file, no sheet 0 found")
	}

	for i := 1; i <= (int(sheet1.MaxRow)); i++ {
		row1 := sheet1.Row(i)

		col1 := row1.Col(0)
		col2 := row1.Col(1)
		col3 := row1.Col(2)
		col4 := row1.Col(3)

		excelDate, err := strconv.ParseFloat(strings.Replace(col1, ",", ".", 1), 64)
		if err != nil {
			return conso, errs.WithEF(err, data.WithField("val", col1), "Failed to read excel date")
		}
		index, err := strconv.Atoi(col2)
		if err != nil {
			return conso, errs.WithEF(err, data.WithField("val", col2), "Failed to read consumption index")
		}
		cons, err := strconv.Atoi(col3)
		if err != nil {
			return conso, errs.WithEF(err, data.WithField("val", col3), "Failed to read day consumption")
		}
		if len(col4) == 0 {
			return conso, errs.WithEF(err, data.WithField("val", col4), "Empty statementt type")
		}
		statementType := StatementType([]rune(col4)[0])
		if !(statementType == Estimated || statementType == Measured) {
			return conso, errs.WithEF(err, data.WithField("val", statementType), "Invalid statement type")
		}

		conso = append(conso, DailyConsumption{
			Day:         timeFromExcelTime(excelDate, false),
			Index:       index,
			Consumption: cons,
			Type:        statementType,
		})
	}

	return conso, nil
}
