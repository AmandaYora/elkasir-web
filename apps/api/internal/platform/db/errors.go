package db

import (
	"errors"

	"github.com/go-sql-driver/mysql"
)

// IsDuplicate true bila error MySQL adalah pelanggaran unique key (1062).
func IsDuplicate(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

// IsForeignKey true bila error MySQL adalah pelanggaran constraint FK (1451/1452).
func IsForeignKey(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && (me.Number == 1451 || me.Number == 1452)
}
