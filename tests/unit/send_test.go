package unit

import (
	"avito/controllers"
	"avito/database"
	"avito/models"
	"bytes"
	"encoding/json"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSendCoin(t *testing.T) {
	sqlDB, db, mock := DbMock(t)
	defer sqlDB.Close()
	var sender models.User
	const defaultCoin = 1000

	sender.ID = 1
	sender.Username = "sender"
	sender.Password = "$2a$14$FRpk.AA8qNdlrSLubQHtPu3i0QTM5j0Hjod7C1MdfEmBrOdWP/Voa"
	sender.Balance = defaultCoin

	var receiver models.User

	receiver.ID = 2
	receiver.Username = "receiver"
	receiver.Password = "$2a$14$8wPpDrk9Xa/RpTzrGU6uZex1m9c6npGUrelYFMzFfeJHcO.EDIgCS"
	receiver.Balance = defaultCoin

	database.PostgresDB = db
	users := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "username", "password", "balance"})
	transactions := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "sender_id", "receiver_id", "amount"})

	t.Run("Should not bind sendCoin schema StatusBadRequest", func(t *testing.T) {
		sendCoinBody := map[string]interface{}{
			"amount": 1000}

		gin.SetMode(gin.TestMode)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		authBody, err := json.Marshal(sendCoinBody)
		assert.NoError(t, err)

		c.Request, _ = http.NewRequest(http.MethodPost, "/", bytes.NewReader(authBody))

		controllers.SendCoin(c)

		if w.Code != http.StatusBadRequest || w.Body.String() != `{"error":"Key: 'SendToPayload.ToUser' Error:Field validation for 'ToUser' failed on the 'required' tag"}` {
			b, _ := ioutil.ReadAll(w.Body)
			t.Error(w.Code, string(b))
		}
		if err = mock.ExpectationsWereMet(); err != nil {
			t.Errorf("There were unfulfilled expectations: %s", err)
		}

	})

	t.Run("Should not authorize due to non-existent user", func(t *testing.T) {
		sendCoinBody := map[string]interface{}{
			"ToUser": "someUser",
			"amount": 10}

		//проверка user'а не пройдет ErrRecordNotFound
		checkUserSQL := `SELECT \* FROM "users" WHERE ID = \$1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT \$2`
		mock.ExpectQuery(checkUserSQL).
			WithArgs(sender.ID, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		gin.SetMode(gin.TestMode)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		authBody, err := json.Marshal(sendCoinBody)
		assert.NoError(t, err)

		c.Request, _ = http.NewRequest(http.MethodPost, "/", bytes.NewReader(authBody))
		c.Set("user_id", sender.ID)

		controllers.SendCoin(c)

		if w.Code != http.StatusUnauthorized || w.Body.String() != `{"error":"Authorization failed"}` {
			b, _ := ioutil.ReadAll(w.Body)
			t.Error(w.Code, string(b))
		}
		if err = mock.ExpectationsWereMet(); err != nil {
			t.Errorf("There were unfulfilled expectations: %s", err)
		}

	})

	t.Run("Trying send negative amount", func(t *testing.T) {
		sendCoinBody := map[string]interface{}{
			"ToUser": "receiver",
			"amount": -100}

		senderAdded := users.AddRow(sender.ID, time.Now(), time.Now(), nil, sender.Username, sender.Password, defaultCoin)
		receiverAdded := users.AddRow(receiver.ID, time.Now(), time.Now(), nil, receiver.Username, receiver.Password, defaultCoin)

		//проверка sender'а
		checkSenderSQL := `SELECT \* FROM "users" WHERE ID = \$1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT \$2`
		mock.ExpectQuery(checkSenderSQL).
			WithArgs(sender.ID, 1).
			WillReturnRows(senderAdded)

		//проверка receiver'а
		checkReceiverSQL := `SELECT \* FROM "users" WHERE Username = \$1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT \$2`
		mock.ExpectQuery(checkReceiverSQL).
			WithArgs(receiver.Username, 1).
			WillReturnRows(receiverAdded)

		//Транзакция: не пройдет так как amount < 0
		createTransactionSQL := `INSERT INTO "transactions" \("created_at","updated_at","deleted_at","sender_id","receiver_id","amount"\) VALUES \(\$1,\$2,\$3,\$4,\$5,\$6\) (.+)`

		mock.ExpectBegin()
		mock.ExpectQuery(createTransactionSQL).WillReturnError(gorm.ErrCheckConstraintViolated)

		gin.SetMode(gin.TestMode)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		authBody, err := json.Marshal(sendCoinBody)
		assert.NoError(t, err)

		c.Request, _ = http.NewRequest(http.MethodPost, "/", bytes.NewReader(authBody))
		c.Set("user_id", sender.ID)

		controllers.SendCoin(c)

		if w.Code != http.StatusBadRequest ||
			w.Body.String() != `{"error":"Incorrect amount of coins to complete the transaction"}` {
			b, _ := ioutil.ReadAll(w.Body)
			t.Error(w.Code, string(b))
		}
		if err = mock.ExpectationsWereMet(); err != nil {
			t.Errorf("There were unfulfilled expectations: %s", err)
		}

	})

	t.Run("Non existing receiver", func(t *testing.T) {
		sendCoinBody := map[string]interface{}{
			"ToUser": "notExistingUser",
			"amount": 10}

		senderAdded := users.AddRow(sender.ID, time.Now(), time.Now(), nil, sender.Username, sender.Password, defaultCoin)

		//проверка sender'а
		checkSenderSQL := `SELECT \* FROM "users" WHERE ID = \$1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT \$2`
		mock.ExpectQuery(checkSenderSQL).
			WithArgs(sender.ID, 1).
			WillReturnRows(senderAdded)

		//проверка receiver'а не пройдет ErrRecordNotFound
		checkReceiverSQL := `SELECT \* FROM "users" WHERE Username = \$1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT \$2`
		mock.ExpectQuery(checkReceiverSQL).
			WillReturnError(gorm.ErrRecordNotFound)

		gin.SetMode(gin.TestMode)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		authBody, err := json.Marshal(sendCoinBody)
		assert.NoError(t, err)

		c.Request, _ = http.NewRequest(http.MethodPost, "/", bytes.NewReader(authBody))
		c.Set("user_id", sender.ID)

		controllers.SendCoin(c)

		if w.Code != http.StatusBadRequest || w.Body.String() != `{"error":"Incorrect receiver's username"}` {
			b, _ := ioutil.ReadAll(w.Body)
			t.Error(w.Code, string(b))
		}
		if err = mock.ExpectationsWereMet(); err != nil {
			t.Errorf("There were unfulfilled expectations: %s", err)
		}

	})

	t.Run("Could not send coins due to insufficient funds", func(t *testing.T) {
		sendCoinBody := map[string]interface{}{
			"ToUser": "receiver",
			"amount": 20000} // больше чем у sender'а

		senderAdded := users.AddRow(sender.ID, time.Now(), time.Now(), nil, sender.Username, sender.Password, defaultCoin)
		receiverAdded := users.AddRow(receiver.ID, time.Now(), time.Now(), nil, receiver.Username, receiver.Password, defaultCoin)

		//проверка sender'а
		checkSenderSQL := `SELECT \* FROM "users" WHERE ID = \$1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT \$2`
		mock.ExpectQuery(checkSenderSQL).
			WithArgs(sender.ID, 1).
			WillReturnRows(senderAdded)

		//проверка receiver'а
		checkReceiverSQL := `SELECT \* FROM "users" WHERE Username = \$1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT \$2`
		mock.ExpectQuery(checkReceiverSQL).
			WithArgs(receiver.Username, 1).
			WillReturnRows(receiverAdded)

		gin.SetMode(gin.TestMode)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		authBody, err := json.Marshal(sendCoinBody)
		assert.NoError(t, err)

		c.Request, _ = http.NewRequest(http.MethodPost, "/", bytes.NewReader(authBody))
		c.Set("user_id", sender.ID)

		controllers.SendCoin(c)

		if w.Code != http.StatusBadRequest || w.Body.String() != `{"error":"Insufficient funds to complete the transaction"}` {
			b, _ := ioutil.ReadAll(w.Body)
			t.Error(w.Code, string(b))
		}
		if err = mock.ExpectationsWereMet(); err != nil {
			t.Errorf("There were unfulfilled expectations: %s", err)
		}

	})

	t.Run("Could not send coins due to broken transaction", func(t *testing.T) {
		sendCoinBody := map[string]interface{}{
			"ToUser": "receiver",
			"amount": 300}

		senderAdded := users.AddRow(sender.ID, time.Now(), time.Now(), nil, sender.Username, sender.Password, defaultCoin)
		receiverAdded := users.AddRow(receiver.ID, time.Now(), time.Now(), nil, receiver.Username, receiver.Password, defaultCoin)

		//проверка sender'а
		checkSenderSQL := `SELECT \* FROM "users" WHERE ID = \$1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT \$2`
		mock.ExpectQuery(checkSenderSQL).
			WithArgs(sender.ID, 1).
			WillReturnRows(senderAdded)

		//проверка receiver'а
		checkReceiverSQL := `SELECT \* FROM "users" WHERE Username = \$1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT \$2`
		mock.ExpectQuery(checkReceiverSQL).
			WithArgs(receiver.Username, 1).
			WillReturnRows(receiverAdded)

		//Транзакция: добавить transaction, обновить баланс у отправителя и получателя, обновить баланс не получилось
		createTransactionSQL := `INSERT INTO "transactions" \("created_at","updated_at","deleted_at","sender_id","receiver_id","amount"\) VALUES \(\$1,\$2,\$3,\$4,\$5,\$6\) (.+)`
		updateBalanceSQL := `UPDATE "users" SET "balance"=\$1,"updated_at"=\$2 WHERE "users"."deleted_at" IS NULL AND "id" = \$3`

		addedTransaction := transactions.AddRow(1, time.Now(), time.Now(), nil, sender.ID, receiver.ID, sendCoinBody["amount"])

		mock.ExpectBegin()
		mock.ExpectQuery(createTransactionSQL).WillReturnRows(addedTransaction)
		mock.ExpectExec(updateBalanceSQL).WillReturnError(gorm.ErrInvalidTransaction)

		gin.SetMode(gin.TestMode)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		authBody, err := json.Marshal(sendCoinBody)
		assert.NoError(t, err)

		c.Request, _ = http.NewRequest(http.MethodPost, "/", bytes.NewReader(authBody))
		c.Set("user_id", sender.ID)

		controllers.SendCoin(c)

		if w.Code != http.StatusInternalServerError || w.Body.String() != `{"error":"Could not send coins"}` {
			b, _ := ioutil.ReadAll(w.Body)
			t.Error(w.Code, string(b))
		}
		if err = mock.ExpectationsWereMet(); err != nil {
			t.Errorf("There were unfulfilled expectations: %s", err)
		}

	})

	t.Run("Should add new transaction OK status", func(t *testing.T) {
		sendCoinBody := map[string]interface{}{
			"ToUser": "receiver",
			"amount": 1000}

		senderAdded := users.AddRow(sender.ID, time.Now(), time.Now(), nil, sender.Username, sender.Password, defaultCoin)
		receiverAdded := users.AddRow(receiver.ID, time.Now(), time.Now(), nil, receiver.Username, receiver.Password, defaultCoin)

		//проверка sender'а
		checkSenderSQL := `SELECT \* FROM "users" WHERE ID = \$1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT \$2`
		mock.ExpectQuery(checkSenderSQL).
			WithArgs(sender.ID, 1).
			WillReturnRows(senderAdded)

		//проверка receiver'а
		checkReceiverSQL := `SELECT \* FROM "users" WHERE Username = \$1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT \$2`
		mock.ExpectQuery(checkReceiverSQL).
			WithArgs(receiver.Username, 1).
			WillReturnRows(receiverAdded)

		//Транзакция: добавить transaction, обновить баланс у отправителя и получателя
		createTransactionSQL := `INSERT INTO "transactions" \("created_at","updated_at","deleted_at","sender_id","receiver_id","amount"\) VALUES \(\$1,\$2,\$3,\$4,\$5,\$6\) (.+)`
		updateBalanceSQL := `UPDATE "users" SET "balance"=\$1,"updated_at"=\$2 WHERE "users"."deleted_at" IS NULL AND "id" = \$3`

		addedTransaction := transactions.AddRow(1, time.Now(), time.Now(), nil, sender.ID, receiver.ID, sendCoinBody["amount"])

		mock.ExpectBegin()
		mock.ExpectQuery(createTransactionSQL).WillReturnRows(addedTransaction)
		mock.ExpectExec(updateBalanceSQL).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(updateBalanceSQL).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		gin.SetMode(gin.TestMode)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		authBody, err := json.Marshal(sendCoinBody)
		assert.NoError(t, err)

		c.Request, _ = http.NewRequest(http.MethodPost, "/", bytes.NewReader(authBody))
		c.Set("user_id", sender.ID)

		controllers.SendCoin(c)

		if w.Code != http.StatusOK {
			b, _ := ioutil.ReadAll(w.Body)
			t.Error(w.Code, string(b))
		}
		if err = mock.ExpectationsWereMet(); err != nil {
			t.Errorf("There were unfulfilled expectations: %s", err)
		}

	})

}
