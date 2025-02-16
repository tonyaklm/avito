package unit

import (
	"avito/controllers"
	"avito/database"
	"bytes"
	"database/sql"
	"encoding/json"
	_ "errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func DbMock(t *testing.T) (*sql.DB, *gorm.DB, sqlmock.Sqlmock) {
	sqldb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	gormdb, err := gorm.Open(postgres.New(postgres.Config{
		Conn: sqldb,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		t.Fatal(err)
	}
	return sqldb, gormdb, mock
}

func TestAuth(t *testing.T) {
	sqlDB, db, mock := DbMock(t)
	defer sqlDB.Close()
	const defaultCoin = 1000

	database.PostgresDB = db
	rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "username", "password", "balance"})

	t.Run("Should not bind user schema StatusBadRequest", func(t *testing.T) {
		user := map[string]interface{}{
			"user":     "admin",
			"password": "admin"}

		gin.SetMode(gin.TestMode)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		authBody, err := json.Marshal(user)
		assert.NoError(t, err)

		c.Request, _ = http.NewRequest(http.MethodPost, "/", bytes.NewReader(authBody))

		controllers.Auth(c)

		if w.Code != http.StatusBadRequest || w.Body.String() != `{"error":"Does not bind schema"}` {
			b, _ := ioutil.ReadAll(w.Body)
			t.Error(w.Code, string(b))
		}
		if err = mock.ExpectationsWereMet(); err != nil {
			t.Errorf("There were unfulfilled expectations: %s", err)
		}

	})
	t.Run("Should not make correct select method", func(t *testing.T) {
		user := map[string]interface{}{
			"username": "admin",
			"password": "admin"}

		expectedSQL := `SELECT \* FROM "users" WHERE Username = \$1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT \$2`
		mock.ExpectQuery(expectedSQL).
			WithArgs(user["username"], 1).
			WillReturnError(gorm.ErrInvalidField)

		gin.SetMode(gin.TestMode)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		authBody, err := json.Marshal(user)
		assert.NoError(t, err)

		c.Request, _ = http.NewRequest(http.MethodPost, "/", bytes.NewReader(authBody))

		controllers.Auth(c)

		if w.Code != http.StatusInternalServerError || w.Body.String() != `{"error":"Could not make search result"}` {
			b, _ := ioutil.ReadAll(w.Body)
			t.Error(w.Code, string(b))
		}
		if err = mock.ExpectationsWereMet(); err != nil {
			t.Errorf("There were unfulfilled expectations: %s", err)
		}

	})

	t.Run("Should not create user because of check constraint violation balance < 0", func(t *testing.T) {
		user := map[string]interface{}{
			"username": "some",
			"password": "some"}

		expectedSQL := `SELECT \* FROM "users" WHERE Username = \$1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT \$2`
		mock.ExpectQuery(expectedSQL).
			WithArgs(user["username"], 1).
			WillReturnError(gorm.ErrRecordNotFound)

		expectedSQL = `INSERT INTO "users" \("created_at","updated_at","deleted_at","username","password","balance"\) VALUES \(\$1,\$2,\$3,\$4,\$5,\$6\) (.+)`
		mock.ExpectBegin()
		mock.ExpectQuery(expectedSQL).WillReturnError(gorm.ErrCheckConstraintViolated)

		gin.SetMode(gin.TestMode)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		authBody, err := json.Marshal(user)
		assert.NoError(t, err)

		c.Request, _ = http.NewRequest(http.MethodPost, "/", bytes.NewReader(authBody))

		controllers.Auth(c)

		if w.Code != http.StatusBadRequest || w.Body.String() != `{"error":"Could not create user"}` {
			b, _ := ioutil.ReadAll(w.Body)
			t.Error(w.Code, string(b))
		}

		if err = mock.ExpectationsWereMet(); err != nil {
			t.Errorf("There were unfulfilled expectations: %s", err)
		}

	})

	t.Run("Successfully register new user", func(t *testing.T) {
		user := map[string]interface{}{
			"username": "admin",
			"password": "admin"}
		hashedPass := "$2a$14$3S5a3omnocQh0KqgOBjjh.dA/TdNRUnaETsLV5PqjrJ/Gs757i8NS"

		//rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "username", "password", "balance"})
		expectedSQL := `SELECT \* FROM "users" WHERE Username = \$1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT \$2`
		mock.ExpectQuery(expectedSQL).
			WithArgs(user["username"], 1).
			WillReturnError(gorm.ErrRecordNotFound)

		expectedSQL = `INSERT INTO "users" \("created_at","updated_at","deleted_at","username","password","balance"\) VALUES \(\$1,\$2,\$3,\$4,\$5,\$6\) (.+)`

		addRow := rows.AddRow(1, time.Now(), time.Now(), nil, user["username"], hashedPass, defaultCoin)
		mock.ExpectBegin()
		mock.ExpectQuery(expectedSQL).WillReturnRows(addRow)
		mock.ExpectCommit()

		gin.SetMode(gin.TestMode)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		authBody, err := json.Marshal(user)
		assert.NoError(t, err)

		c.Request, _ = http.NewRequest(http.MethodPost, "/", bytes.NewReader(authBody))

		controllers.Auth(c)

		if w.Code != http.StatusOK {
			b, _ := ioutil.ReadAll(w.Body)
			t.Error(w.Code, string(b))
		}
		if err = mock.ExpectationsWereMet(); err != nil {
			t.Errorf("There were unfulfilled expectations: %s", err)
		}

	})
	t.Run("Successfully auth old user", func(t *testing.T) {
		user := map[string]interface{}{
			"username": "admin",
			"password": "admin"}
		hashedPass := "$2a$14$3S5a3omnocQh0KqgOBjjh.dA/TdNRUnaETsLV5PqjrJ/Gs757i8NS"
		addRow := rows.AddRow(1, time.Now(), time.Now(), nil, user["username"], hashedPass, defaultCoin)

		// успешный select

		expectedSQL := `SELECT \* FROM "users" WHERE Username = \$1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT \$2`
		mock.ExpectQuery(expectedSQL).
			WithArgs(user["username"], 1).
			WillReturnRows(addRow)

		gin.SetMode(gin.TestMode)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		authBody, err := json.Marshal(user)
		assert.NoError(t, err)

		c.Request, _ = http.NewRequest(http.MethodPost, "/", bytes.NewReader(authBody))

		controllers.Auth(c)

		if w.Code != http.StatusOK {
			b, _ := ioutil.ReadAll(w.Body)
			t.Error(w.Code, string(b))
		}

		if err = mock.ExpectationsWereMet(); err != nil {
			t.Errorf("There were unfulfilled expectations: %s", err)
		}

	})

}
