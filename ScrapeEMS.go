package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/simplereach/timeutils"
)

const (
	path        = "/einsatznachbearbeitungberichtbycommonds/berichtliststandarddaten"
	querystring = "&mDataProp_0=elrEinsatzNummer&mDataProp_1=formularBoolean1&mDataProp_2=einsatzDatum&mDataProp_3=berichtsart&mDataProp_4=string100n1&mDataProp_5=statusBearbeitung&mDataProp_6=statusVerrechnung&mDataProp_7=string100n3&mDataProp_8=resRechte_kurzzeichen&mDataProp_9=string100n2&mDataProp_10=string100n4&mDataProp_11=string100n5&mDataProp_12=string100n6&mDataProp_13=naechsterBearbeiter_loginname&sSearch=&bRegex=false&sSearch_0=&bRegex_0=false&bSearchable_0=true&sSearch_1=&bRegex_1=false&bSearchable_1=true&sSearch_2=&bRegex_2=false&bSearchable_2=true&sSearch_3=&bRegex_3=false&bSearchable_3=true&sSearch_4=&bRegex_4=false&bSearchable_4=true&sSearch_5=&bRegex_5=false&bSearchable_5=true&sSearch_6=&bRegex_6=false&bSearchable_6=true&sSearch_7=&bRegex_7=false&bSearchable_7=true&sSearch_8=&bRegex_8=false&bSearchable_8=true&sSearch_9=&bRegex_9=false&bSearchable_9=true&sSearch_10=&bRegex_10=false&bSearchable_10=true&sSearch_11=&bRegex_11=false&bSearchable_11=true&sSearch_12=&bRegex_12=false&bSearchable_12=true&sSearch_13=&bRegex_13=false&bSearchable_13=true&iSortingCols=1&iSortCol_0=2&sSortDir_0=desc&bSortable_0=true&bSortable_1=true&bSortable_2=true&bSortable_3=true&bSortable_4=true&bSortable_5=true&bSortable_6=true&bSortable_7=true&bSortable_8=true&bSortable_9=true&bSortable_10=true&bSortable_11=true&bSortable_12=true&bSortable_13=true"
)

var (
	protocol   = "https://"
	emsPort    = ":443"
	emsURL     = ""
	kontoId    = ""
	menueId    = ""
	username   = ""
	password   = ""
	outputtype = ""
	csvdel     = ";"
	count      = 10
)

type Scraper struct {
	Client *http.Client
}

type AuthenticityToken struct {
	Token string
}

type Project struct {
	Name string
}

type EinsatzBerichte struct {
	Einsatz []struct {
		RowID                string         `json:"DT_RowId"`
		Einsatzkategorie     string         `json:"berichtsart"`
		Einheit              string         `json:"einheit"`
		EinsatzDatum         timeutils.Time `json:"einsatzDatum"`
		ElrEinsatzNummer     string         `json:"elrEinsatzNummer"`
		FormularBoolean1     string         `json:"formularBoolean1"`
		NaechsterBearbeiter  string         `json:"naechsterBearbeiter_loginname"`
		ResRechteKurzzeichen string         `json:"resRechte_kurzzeichen"`
		StatusBearbeitung    string         `json:"statusBearbeitung"`
		StatusVerrechnung    string         `json:"statusVerrechnung"`
		Einsatzart           string         `json:"string100n1"`
		PLZ                  string         `json:"string100n2"`
		OrtKurz              string         `json:"string100n3"`
		OrtLang              string         `json:"string100n4"`
		Strasse              string         `json:"string100n5"`
		Sonstige             string         `json:"string100n6"`
	} `json:"aaData"`
	ITotalDisplayRecords int `json:"iTotalDisplayRecords"`
	ITotalRecords        int `json:"iTotalRecords"`
	SEcho                int `json:"sEcho"`
}

func (app *Scraper) getToken() AuthenticityToken {
	loginURL := protocol + emsURL + emsPort + "/login"
	client := app.Client

	response, err := client.Get(loginURL)

	if err != nil {
		log.Fatalln("Error fetching response. ", err)
	}

	defer response.Body.Close()

	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		log.Fatal("Error loading HTTP response body. ", err)
	}

	token, _ := document.Find("input[name='authenticityToken']").Attr("value")

	authenticityToken := AuthenticityToken{
		Token: token,
	}

	return authenticityToken
}

func (app *Scraper) login() (string, error) {
	client := app.Client

	authenticityToken := app.getToken()

	loginURL := protocol + emsURL + emsPort + "/login"

	data := url.Values{
		"authenticityToken": {authenticityToken.Token},
		"username":          {username},
		"password":          {password},
	}

	response, err := client.PostForm(loginURL, data)

	if err != nil {
		return "", err
	}

	defer response.Body.Close()

	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return "", err
	}

	kontoStr, _ := document.Find("li #3130 a").Attr("href")
	kontoMap := strings.Split(kontoStr, "/")

	if len(kontoMap) < 3 {
		return "", fmt.Errorf("Account ID could not be found")
	}

	return kontoMap[2], nil
}

func (app *Scraper) getOperations(target interface{}) error {
	projectsURL := fmt.Sprintf("%v%v%v%v?konto=%v&menueId=%v&sEcho=1&iColumns=14&sColumns=&iDisplayStart=0&iDisplayLength=%d%v", protocol, emsURL, emsPort, path, kontoId, menueId, count, querystring)

	client := app.Client

	response, err := client.Get(projectsURL)

	if err != nil {
		return fmt.Errorf("Error fetching response ", err)
	}

	defer response.Body.Close()

	return json.NewDecoder(response.Body).Decode(target)
}

func main() {
	var err error
	menueId = "3130"

	flag.StringVar(&emsURL, "url", "", "IP/Domain of the ELDIS-Management-Suite")
	flag.StringVar(&username, "user", "", "your EMS username")
	flag.StringVar(&password, "pass", "", "your EMS password")
	flag.StringVar(&outputtype, "output", "json", "data type/format of the output")
	flag.StringVar(&csvdel, "del", ";", "separator when exporting as CSV")
	flag.IntVar(&count, "count", 10, "number of entries to be queried")

	flag.Parse()

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	jar, _ := cookiejar.New(nil)

	app := Scraper{
		Client: &http.Client{Jar: jar},
	}

	kontoId, err = app.login()

	if err != nil {
		log.Fatal("Error on login: ", err)
		return
	}

	berichte := EinsatzBerichte{}
	err = app.getOperations(&berichte)

	if err != nil {
		log.Fatal("Error on extracting: ", err)
		return
	}

	switch outputtype {
	case "json":
		jsonStr, _ := json.Marshal(berichte)
		fmt.Println(string(jsonStr))
	case "csv":
		bomUtf8 := []byte{0xEF, 0xBB, 0xBF}
		fmt.Print(string(bomUtf8[:]))
		fmt.Println(strings.Join([]string{
			"Kategorie",
			"Einheit",
			"Datum",
			"EinsatzNummer",
			"Status",
			"Einsatzart",
			"PLZ",
			"Ort",
			"Strasse",
		}, csvdel))

		for row := range berichte.Einsatz {
			einsatz := berichte.Einsatz[row]
			fmt.Println(strings.Join([]string{
				einsatz.Einsatzkategorie,
				einsatz.Einheit,
				einsatz.EinsatzDatum.Format("2006-01-02 15:04:05"),
				einsatz.ElrEinsatzNummer,
				einsatz.StatusBearbeitung,
				einsatz.Einsatzart,
				einsatz.PLZ,
				einsatz.OrtLang,
				einsatz.Strasse,
			}, csvdel))
		}
	default:
		fmt.Printf("%#v\n", berichte)
	}
}
