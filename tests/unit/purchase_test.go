package unit

import (
	"avito/controllers"
	"avito/database"
	"avito/models"
	_ "errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBuyItem(t *testing.T) {
	sqlDB, db, mock := DbMock(t)
	defer sqlDB.Close()
	var user models.User
	const defaultCoin = 1000

	user.ID = 1
	user.Username = "admin"
	user.Password = "$2a$14$3S5a3omnocQh0KqgOBjjh.dA/TdNRUnaETsLV5PqjrJ/Gs757i8NS"
	user.Balance = defaultCoin

	var item models.Item
	item.ID = 1
	item.ItemName = "t-shirt"
	item.Price = 80

	database.PostgresDB = db
	users := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "username", "password", "balance"})
	purchases := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "item_id", "user_id", "price"})
	items := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "item_name", "price"})

	t.Run("Should not authorize due to wrong token", func(t *testing.T) {

		gin.SetMode(gin.TestMode)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)

		c.Params = []gin.Param{gin.Param{Key: "item", Value: item.ItemName}}

		controllers.BuyItem(c)

		if w.Code != http.StatusUnauthorized || w.Body.String() != `{"error":"Authorization failed"}` {
			b, _ := ioutil.ReadAll(w.Body)
			t.Error(w.Code, string(b))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("There were unfulfilled expectations: %s", err)
		}

	})
	t.Run("Should not authorize due to non-existent user", func(t *testing.T) {
		//проверка user'а не пройдет ErrRecordNotFound
		checkUserSQL := `SELECT \* FROM "users" WHERE ID = \$1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT \$2`
		mock.ExpectQuery(checkUserSQL).
			WithArgs(user.ID, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		gin.SetMode(gin.TestMode)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
		c.Set("user_id", user.ID)

		c.Params = []gin.Param{gin.Param{Key: "item", Value: item.ItemName}}

		controllers.BuyItem(c)

		if w.Code != http.StatusUnauthorized || w.Body.String() != `{"error":"Authorization failed"}` {
			b, _ := ioutil.ReadAll(w.Body)
			t.Error(w.Code, string(b))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("There were unfulfilled expectations: %s", err)
		}

	})

	t.Run("Should return 400 due to not passed item", func(t *testing.T) {
		addedUser := users.AddRow(user.ID, time.Now(), time.Now(), nil, user.Username, user.Password, defaultCoin)

		//проверка user'а
		checkUserSQL := `SELECT \* FROM "users" WHERE ID = \$1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT \$2`
		mock.ExpectQuery(checkUserSQL).
			WithArgs(user.ID, 1).
			WillReturnRows(addedUser)

		//проверка item'а ErrRecordNotFound
		checkItemSQL := `SELECT \* FROM "items" WHERE item_name = \$1 AND "items"."deleted_at" IS NULL ORDER BY "items"."id" LIMIT \$2`
		mock.ExpectQuery(checkItemSQL).
			WithArgs("", 1).
			WillReturnError(gorm.ErrRecordNotFound)

		gin.SetMode(gin.TestMode)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
		c.Set("user_id", user.ID)
		controllers.BuyItem(c)

		if w.Code != http.StatusBadRequest || w.Body.String() != `{"error":"Could not find item"}` {
			b, _ := ioutil.ReadAll(w.Body)
			t.Error(w.Code, string(b))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("There were unfulfilled expectations: %s", err)
		}

	})
	t.Run("Should return 400 due to not enough money", func(t *testing.T) {
		addedUser := users.AddRow(user.ID, time.Now(), time.Now(), nil, user.Username, user.Password, item.Price-1)
		addedItem := items.AddRow(item.ID, time.Now(), time.Now(), nil, item.ItemName, item.Price)

		//проверка user'а
		checkUserSQL := `SELECT \* FROM "users" WHERE ID = \$1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT \$2`
		mock.ExpectQuery(checkUserSQL).
			WithArgs(user.ID, 1).
			WillReturnRows(addedUser)

		//проверка item'а
		checkItemSQL := `SELECT \* FROM "items" WHERE item_name = \$1 AND "items"."deleted_at" IS NULL ORDER BY "items"."id" LIMIT \$2`
		mock.ExpectQuery(checkItemSQL).
			WithArgs(item.ItemName, 1).
			WillReturnRows(addedItem)

		gin.SetMode(gin.TestMode)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
		c.Set("user_id", user.ID)

		c.Params = []gin.Param{gin.Param{Key: "item", Value: item.ItemName}}

		controllers.BuyItem(c)

		if w.Code != http.StatusBadRequest || w.Body.String() != `{"error":"Insufficient funds to complete the transaction"}` {
			b, _ := ioutil.ReadAll(w.Body)
			t.Error(w.Code, string(b))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("There were unfulfilled expectations: %s", err)
		}

	})

	t.Run("Could not buy item due to broken transaction", func(t *testing.T) {
		addedUser := users.AddRow(user.ID, time.Now(), time.Now(), nil, user.Username, user.Password, defaultCoin)
		addedItem := items.AddRow(item.ID, time.Now(), time.Now(), nil, item.ItemName, item.Price)

		//проверка user'а
		checkUserSQL := `SELECT \* FROM "users" WHERE ID = \$1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT \$2`
		mock.ExpectQuery(checkUserSQL).
			WithArgs(user.ID, 1).
			WillReturnRows(addedUser)

		//проверка item'а
		checkItemSQL := `SELECT \* FROM "items" WHERE item_name = \$1 AND "items"."deleted_at" IS NULL ORDER BY "items"."id" LIMIT \$2`
		mock.ExpectQuery(checkItemSQL).
			WithArgs(item.ItemName, 1).
			WillReturnRows(addedItem)

		//транзация покупки и изменение баланса не пройдет
		purchaseSQL := `INSERT INTO "purchases" \("created_at","updated_at","deleted_at","item_id","user_id","price"\) VALUES \(\$1,\$2,\$3,\$4,\$5,\$6\) (.+)`
		updateBalanceSQL := `UPDATE "users" SET "balance"=\$1,"updated_at"=\$2 WHERE "users"."deleted_at" IS NULL AND "id" = \$3`

		addedPurchase := purchases.AddRow(1, time.Now(), time.Now(), nil, item.ID, user.ID, item.Price)

		mock.ExpectBegin()
		mock.ExpectQuery(purchaseSQL).WillReturnRows(addedPurchase)
		mock.ExpectExec(updateBalanceSQL).WillReturnError(gorm.ErrInvalidTransaction)
		// не ожидается коммит

		gin.SetMode(gin.TestMode)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
		c.Set("user_id", user.ID)

		c.Params = []gin.Param{gin.Param{Key: "item", Value: item.ItemName}}

		controllers.BuyItem(c)

		if w.Code != http.StatusInternalServerError || w.Body.String() != `{"error":"Could not make a transaction"}` {
			b, _ := ioutil.ReadAll(w.Body)
			t.Error(w.Code, string(b))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("There were unfulfilled expectations: %s", err)
		}

	})

	t.Run("Should add new purchase t-shirt OK status", func(t *testing.T) {
		addedUser := users.AddRow(user.ID, time.Now(), time.Now(), nil, user.Username, user.Password, defaultCoin)
		addedItem := items.AddRow(item.ID, time.Now(), time.Now(), nil, item.ItemName, item.Price)

		//проверка user'а
		checkUserSQL := `SELECT \* FROM "users" WHERE ID = \$1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT \$2`
		mock.ExpectQuery(checkUserSQL).
			WithArgs(user.ID, 1).
			WillReturnRows(addedUser)

		//проверка item'а
		checkItemSQL := `SELECT \* FROM "items" WHERE item_name = \$1 AND "items"."deleted_at" IS NULL ORDER BY "items"."id" LIMIT \$2`
		mock.ExpectQuery(checkItemSQL).
			WithArgs(item.ItemName, 1).
			WillReturnRows(addedItem)

		//транзация покупки и изменение баланса
		purchaseSQL := `INSERT INTO "purchases" \("created_at","updated_at","deleted_at","item_id","user_id","price"\) VALUES \(\$1,\$2,\$3,\$4,\$5,\$6\) (.+)`
		updateBalanceSQL := `UPDATE "users" SET "balance"=\$1,"updated_at"=\$2 WHERE "users"."deleted_at" IS NULL AND "id" = \$3`

		addedPurchase := purchases.AddRow(1, time.Now(), time.Now(), nil, item.ID, user.ID, item.Price)

		mock.ExpectBegin()
		mock.ExpectQuery(purchaseSQL).WillReturnRows(addedPurchase)
		mock.ExpectExec(updateBalanceSQL).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		gin.SetMode(gin.TestMode)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
		c.Set("user_id", user.ID)

		c.Params = []gin.Param{gin.Param{Key: "item", Value: item.ItemName}}

		controllers.BuyItem(c)

		if w.Code != http.StatusOK {
			b, _ := ioutil.ReadAll(w.Body)
			t.Error(w.Code, string(b))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("There were unfulfilled expectations: %s", err)
		}

	})

}
