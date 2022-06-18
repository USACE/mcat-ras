package pgdb

import (
	"fmt"
	"os"

	_ "github.com/jackc/pgx/stdlib"
	"github.com/jmoiron/sqlx"
)

func DBInit() *sqlx.DB {

	creds := fmt.Sprintf("user=%s password=%s host=%s port=%s database=%s sslmode=disable",
		os.Getenv("DBUSER"), os.Getenv("DBPASS"), os.Getenv("DBHOST"), os.Getenv("DBPORT"), os.Getenv("DBNAME"))

	return sqlx.MustOpen("pgx", creds)
}
