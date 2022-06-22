package mysql

import (
	"context"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/jmoiron/sqlx"

	// mysql driver
	_ "github.com/go-sql-driver/mysql"
)

// MySQL is implement datastore in MySQL
type MySQL struct {
	Conn *sqlx.DB
}

var cacheConnectionID = 0

// New create mysql connection
func New(dsn string) (*MySQL, error) {
	u, err := getMySQLURL(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to get MySQL URL: %w", err)
	}

	conn, err := sqlx.Open("mysql", u)
	if err != nil {
		return nil, fmt.Errorf("failed to create mysql connection: %w", err)
	}

	return &MySQL{
		Conn: conn,
	}, nil
}

func getMySQLURL(dsn string) (string, error) {
	c, err := mysql.ParseDSN(dsn)
	if err != nil {
		return "", fmt.Errorf("failed to parse DSN: %w", err)
	}

	c.Loc = time.UTC
	c.ParseTime = true
	c.Collation = "utf8mb4_general_ci"
	if c.Params == nil {
		c.Params = map[string]string{}
	}
	c.Params["sql_mode"] = "'TRADITIONAL,NO_AUTO_VALUE_ON_ZERO,ONLY_FULL_GROUP_BY'"

	c.InterpolateParams = true

	return c.FormatDSN(), nil
}

func (m *MySQL) getConnectionID(ctx context.Context) (int, error) {
	if cacheConnectionID != 0 {
		return cacheConnectionID, nil
	}

	var cID int
	if err := m.Conn.GetContext(ctx, &cID, `SELECT CONNECTION_ID()`); err != nil {
		return -1, fmt.Errorf("failed to execute SELECT query: %w", err)
	}
	cacheConnectionID = cID

	return cacheConnectionID, nil
}
