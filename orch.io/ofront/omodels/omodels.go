package omodels

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/OKESTRO-AIDevOps/nkia/orch.io/ofront/omodules"
	modules "github.com/OKESTRO-AIDevOps/nkia/pkg/challenge"
	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

type OrchestratorRecord_Pubkey struct {
	pubkey string
}

type OrchestratorRecord_Email struct {
	email string
}

type OrchestratorRecord_RequestKey struct {
	request_key string
}

func DbEstablish(db_id string, db_pw string, db_host string, db_name string) {

	db_info := fmt.Sprintf("%s:%s@tcp(%s)/%s", db_id, db_pw, db_host, db_name)

	DB, _ = sql.Open("mysql", db_info)

	DB.SetConnMaxLifetime(time.Second * 10)
	DB.SetConnMaxIdleTime(time.Second * 5)
	DB.SetMaxOpenConns(10)
	DB.SetMaxIdleConns(10)

	fmt.Println("DB Connected")

}

func DbQuery(query string, args []any) (*sql.Rows, error) {

	var empty_row *sql.Rows

	results, err := DB.Query(query, args[0:]...)

	if err != nil {

		return empty_row, err

	}

	return results, err

}

func FrontAccessAuth(session_id string) (string, error) {

	var request_key string

	var result_container_request_key []OrchestratorRecord_RequestKey

	q := "SELECT request_key FROM orchestrator_record WHERE osid = ?"

	a := []any{session_id}

	res, err := DbQuery(q, a)

	if err != nil {
		return "", fmt.Errorf("failed to get access: %s", err.Error())
	}

	for res.Next() {

		var or OrchestratorRecord_RequestKey

		err = res.Scan(&or.request_key)

		if err != nil {

			return "", fmt.Errorf("failed to register: %s", err.Error())

		}

		result_container_request_key = append(result_container_request_key, or)

	}

	if len(result_container_request_key) != 1 {
		return "", fmt.Errorf("failed to get access: %s", "duplicate")
	}

	request_key = result_container_request_key[0].request_key

	res.Close()

	return request_key, nil
}

func RegisterOsidAndRequestKeyByOAuth(session_id string, oauth_struct omodules.OAuthStruct) (string, error) {

	var request_key string

	var result_container_email []OrchestratorRecord_Email

	var result_container_request_key []OrchestratorRecord_RequestKey

	q := "SELECT email FROM orchestrator_record WHERE email = ?"

	a := []any{oauth_struct.EMAIL}

	res, err := DbQuery(q, a)

	if err != nil {
		return "", fmt.Errorf("failed to register: %s", err.Error())
	}

	for res.Next() {

		var or OrchestratorRecord_Email

		err = res.Scan(&or.email)

		if err != nil {

			return "", fmt.Errorf("failed to register: %s", err.Error())

		}

		result_container_email = append(result_container_email, or)

	}

	res.Close()

	if len(result_container_email) == 0 {

		if err := RegisterNewRecord(oauth_struct.EMAIL); err != nil {

			return "", fmt.Errorf("failed to register: %s", err.Error())
		}
	}

	request_key, err = modules.RandomHex(16)

	if err != nil {
		return "", fmt.Errorf("failed to register: %s", err.Error())
	}

	q = "UPDATE orchestrator_record SET osid = ?, request_key =? WHERE email = ?"

	a = []any{session_id, request_key, oauth_struct.EMAIL}

	res, err = DbQuery(q, a)

	if err != nil {
		return "", fmt.Errorf("failed to register: %s", err.Error())
	}

	res.Close()

	q = "SELECT request_key FROM orchestrator_record WHERE email = ?"

	a = []any{oauth_struct.EMAIL}

	res, err = DbQuery(q, a)

	if err != nil {
		return "", fmt.Errorf("failed to register: %s", err.Error())
	}

	for res.Next() {

		var or OrchestratorRecord_RequestKey

		err = res.Scan(&or.request_key)

		if err != nil {

			return "", fmt.Errorf("failed to register: %s", err.Error())

		}

		result_container_request_key = append(result_container_request_key, or)

	}

	if len(result_container_request_key) != 1 {
		return "", fmt.Errorf("failed to register: %s", "duplicate")
	}

	request_key = result_container_request_key[0].request_key

	res.Close()

	return request_key, nil

}

func RegisterOsidAndRequestKeyByEmail(email string, session_id string) (string, error) {

	var request_key string

	var result_container_email []OrchestratorRecord_Email

	var result_container_request_key []OrchestratorRecord_RequestKey

	q := "SELECT email FROM orchestrator_record WHERE email = ?"

	a := []any{email}

	res, err := DbQuery(q, a)

	if err != nil {
		return "", fmt.Errorf("failed to register: %s", err.Error())
	}

	for res.Next() {

		var or OrchestratorRecord_Email

		err = res.Scan(&or.email)

		if err != nil {

			return "", fmt.Errorf("failed to register: %s", err.Error())

		}

		result_container_email = append(result_container_email, or)

	}

	res.Close()

	if len(result_container_email) != 1 {

		return "", fmt.Errorf("failed to register: %s", "length")

	}

	request_key, err = modules.RandomHex(16)

	if err != nil {
		return "", fmt.Errorf("failed to register: %s", err.Error())
	}

	q = "UPDATE orchestrator_record SET osid = ?, request_key =? WHERE email = ?"

	a = []any{session_id, request_key, email}

	res, err = DbQuery(q, a)

	if err != nil {
		return "", fmt.Errorf("failed to register: %s", err.Error())
	}

	res.Close()

	q = "SELECT request_key FROM orchestrator_record WHERE email = ?"

	a = []any{email}

	res, err = DbQuery(q, a)

	if err != nil {
		return "", fmt.Errorf("failed to register: %s", err.Error())
	}

	for res.Next() {

		var or OrchestratorRecord_RequestKey

		err = res.Scan(&or.request_key)

		if err != nil {

			return "", fmt.Errorf("failed to register: %s", err.Error())

		}

		result_container_request_key = append(result_container_request_key, or)

	}

	if len(result_container_request_key) != 1 {
		return "", fmt.Errorf("failed to register: %s", "duplicate")
	}

	request_key = result_container_request_key[0].request_key

	res.Close()

	return request_key, nil

}

func RegisterNewRecord(email string) error {

	q := "INSERT INTO orchestrator_record(email) VALUES (?)"

	a := []any{email}

	res, err := DbQuery(q, a)

	if err != nil {
		return fmt.Errorf("failed to register new record: %s", err.Error())
	}

	res.Close()

	return nil

}

func GetPubkeyByEmail(email string) (string, error) {

	var ret string

	var result_container_pubkey []OrchestratorRecord_Pubkey

	q := "SELECT pubkey FROM orchestrator_record WHERE email = ?"

	a := []any{email}

	res, err := DbQuery(q, a)

	if err != nil {
		return "", fmt.Errorf("failed to get pubkey: %s", err.Error())
	}

	for res.Next() {

		var or OrchestratorRecord_Pubkey

		err = res.Scan(&or.pubkey)

		if err != nil {

			return "", fmt.Errorf("failed to get pubkey: %s", err.Error())

		}

		result_container_pubkey = append(result_container_pubkey, or)

	}

	res.Close()

	if len(result_container_pubkey) != 1 {
		return "", fmt.Errorf("failed to get pubkey: %s", "length")
	}

	ret = result_container_pubkey[0].pubkey

	return ret, nil
}
