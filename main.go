package main

import (
	"fmt"
	"os"

	"database/sql"
	"time"

	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/dcu/go-authy"
	_ "github.com/go-sql-driver/mysql"
)

var (
	//All global environment variables should be set at the beginning of the application, then remain unchanged.
	authyAPIKey   = os.Getenv("STREETSWEEP_AUTHY_API_KEY")
	Port          = os.Getenv("PORT")
	MySQLPassword = os.Getenv("MYSQL_PASSWORD")
	authyAPI      *authy.Authy
	DB            *sql.DB
)

type StartVerifyReq struct {
	Via         string `json:"via"`
	CountryCode int    `json:"country_code"`
	PhoneNumber int    `json:"phone_number"`
}

//{"via":"sms","timezone":"America/Phoenix","nth_day":"second","weekday":"friday","country_code":"1","phone_number":"8054234224","token":"3"}

type VerifyCodeReq struct {
	Timezone    string `json:"timezone"`
	NthDay      int    `json:"nth_day"`
	Weekday     int    `json:"weekday"`
	CountryCode int    `json:"country_code"`
	PhoneNumber int    `json:"phone_number"`
	Token       string `json:"token"`
}

func init() {
	if authyAPIKey == "" {
		log.Fatal("STREETSWEEP_AUTHY_API_KEY environment variable not set")
	}
	authyAPI = authy.NewAuthyAPI(authyAPIKey)

	db, err := sql.Open("mysql",
		"root:"+MySQLPassword+"@tcp(127.0.0.1:3306)/hello")
	if err != nil {
		log.Fatal(err)
	}

	DB = db

	err = DB.Ping()
	if err != nil {
		log.Fatal(err)
		// do something here
	}

	dropTableCommand := `drop table if exists alerts`
	_, err = DB.Exec(dropTableCommand)

	if err != nil {
		log.Fatal(err)
	}

	createTableCommand := `CREATE TABLE IF NOT EXISTS alerts(
				   ID INT NOT NULL AUTO_INCREMENT,
				   PHONE_NUMBER CHAR(10) NOT NULL,
				   COUNTRY_CODE INT NOT NULL,
				   NTH_DAY INT NOT NULL,
				   TIMEZONE VARCHAR(100) NOT NULL,
				   WEEKDAY VARCHAR(20) NOT NULL,
				   NEXT_CALL BIGINT NOT NULL,
				   PRIMARY KEY  (ID)
				)`
	_, err = DB.Exec(createTableCommand)

	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	go func() {
		stmt, err := DB.Prepare("select * from alerts where NEXT_CALL < ?")
		if err != nil {
			log.Fatal(err)
		}
		defer stmt.Close()
		for range time.Tick(1 * time.Second) {
			go func() {
				nowUTC := time.Now().Unix()
				rows, err := stmt.Query(nowUTC)
				if err != nil {
					log.Fatal(err)
				}
				defer rows.Close()
				for rows.Next() {
					//alert :=
					//err := rows.Scan(&id, &name)
					//if err != nil {
					//	log.Fatal(err)
					//}
					log.Println("bob")
				}
				if err = rows.Err(); err != nil {
					log.Fatal(err)
				}
			}()
		}
	}()

	http.Handle("/", http.FileServer(http.Dir("./public")))
	http.HandleFunc("/verification/start", verificationStartHandler)
	http.HandleFunc("/verification/verify", verificationVerifyHandler)
	log.Println("Magic happening on port " + Port)
	log.Fatal(http.ListenAndServe(":"+Port, nil))
}

func verificationStartHandler(w http.ResponseWriter, r *http.Request) {
	//requestDump, err := httputil.DumpRequest(r, true)
	//if err != nil {
	//	fmt.Println(err)
	//}
	//fmt.Println("1", string(requestDump))

	decoder := json.NewDecoder(r.Body)
	var t StartVerifyReq
	err := decoder.Decode(&t)
	if err != nil {
		panic(err)
	}
	defer r.Body.Close()
	log.Printf("bob: %+v", t)

	verification, err := authyAPI.StartPhoneVerification(t.CountryCode, strconv.Itoa(t.PhoneNumber), t.Via, url.Values{})
	if verification.Success {
		w.WriteHeader(http.StatusOK)
	} else {
		//todo: do this better. figure out all the ways that start phone verification could fail and handle all of them well
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, "validation code incorrect")
	}
	fmt.Println("verification, err", verification, err)
}

func verificationVerifyHandler(w http.ResponseWriter, r *http.Request) {
	//requestDump, err := httputil.DumpRequest(r, true)
	//if err != nil {
	//	fmt.Println(err)
	//}
	//fmt.Println("2", string(requestDump))

	decoder := json.NewDecoder(r.Body)
	var t VerifyCodeReq
	err := decoder.Decode(&t)
	if err != nil {
		panic(err)
	}
	defer r.Body.Close()
	log.Printf("bob: %+v", t)

	verification, err := authyAPI.CheckPhoneVerification(t.CountryCode, strconv.Itoa(t.PhoneNumber), t.Token, url.Values{})
	fmt.Println("verification, err", verification, err)

	if verification.Success {
		w.WriteHeader(200)
		save(t)
	} else {
		//todo: do this better. figure out all the ways that CheckPhoneVerification could fail and handle all of them well
		w.WriteHeader(401)
		io.WriteString(w, "validation code incorrect")
	}
}

func save(alert VerifyCodeReq) {
	stmt, err := DB.Prepare("INSERT INTO alerts (PHONE_NUMBER, COUNTRY_CODE, NTH_DAY, TIMEZONE, WEEKDAY, NEXT_CALL) VALUES (?,?,?,?,?,?)")
	if err != nil {
		log.Fatal(err) //todo: change these log.fatals to a more reasonable error handling
	}

	nextCall, err := CalculateNextCall(alert)
	if err != nil {
		log.Fatal(err)
	}

	_, err = stmt.Exec(alert.PhoneNumber, alert.CountryCode, alert.NthDay, alert.Timezone, alert.Weekday, nextCall)
	if err != nil {
		log.Fatal(err)
	}
}

var Now = func() time.Time {
	return time.Now()
}

func CalculateNextCall(alert VerifyCodeReq) (int64, error) {
	var NextCallTime int64

	location, err := time.LoadLocation(alert.Timezone)
	if err != nil {
		return NextCallTime, err
	}

	now := Now().In(location)
	timeAtNthDayOfMonth := TimeAtNthDayOfMonth(now, alert.NthDay, alert.Weekday, 19) //todo: change this hard coded hour
	if now.After(timeAtNthDayOfMonth) {
		timeAtNthDayOfMonth = TimeAtNthDayOfMonth(now.AddDate(0, 1, 0), alert.NthDay, alert.Weekday, 19)
	}

	NextCallTime = timeAtNthDayOfMonth.Unix()

	return NextCallTime, nil
}

func TimeAtNthDayOfMonth(t time.Time, nthDay int, weekday int, hour int) time.Time {
	firstDayOfThisMonth := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	dateOfFirstWeekday := ((weekday+7)-int(firstDayOfThisMonth.Weekday()))%7 + 1
	dateOfNthWeekday := dateOfFirstWeekday + ((nthDay - 1) * 7)
	TimeAtNthDayOfMonth := time.Date(t.Year(), t.Month(), dateOfNthWeekday, hour, 0, 0, 0, t.Location())
	return TimeAtNthDayOfMonth
}
