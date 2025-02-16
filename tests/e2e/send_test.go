package e2e

import (
	"avito/controllers"
	"bytes"
	"encoding/json"
	"github.com/go-playground/assert/v2"
	"github.com/stretchr/testify/require"
	"math/rand"
	"net/http"
	"strconv"
	"testing"
	"time"
)

func sendCoins(t *testing.T, sendBody []byte, token string, targetStatus int) {
	const sendUrl = "http://localhost:8080/api/sendCoin"

	req, err := http.NewRequest(http.MethodPost, sendUrl, bytes.NewBuffer(sendBody))
	require.NoError(t, err)

	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", token)

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	res, err := client.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, targetStatus, res.StatusCode)
}

func TestUnauthorizedTransaction(t *testing.T) {
	invalidToken := "not the actual token"

	sendCoinSchema := map[string]interface{}{
		"toUser": "user",
		"amount": 500}
	senBody, err := json.Marshal(sendCoinSchema)
	require.NoError(t, err)

	sendCoins(t, senBody, invalidToken, http.StatusUnauthorized)
}

func TestNoRecipientTransaction(t *testing.T) {
	senderUser := map[string]interface{}{
		"username": "Katy",
		"password": "password"}
	sendCoinSchema := map[string]interface{}{
		"toUser": "noSuchUser",
		"amount": 100}

	authBody, err := json.Marshal(senderUser)
	require.NoError(t, err)

	senderToken := authUser(t, authBody, http.StatusOK, true)

	senBody, err := json.Marshal(sendCoinSchema)
	require.NoError(t, err)

	sendCoins(t, senBody, senderToken.SignedToken, http.StatusBadRequest)
}

func TestNotEnoughFundsTransaction(t *testing.T) {
	senderUser := map[string]interface{}{
		"username": "sender",
		"password": "senderPassword"}

	receiverUser := map[string]interface{}{
		"username": "receiver",
		"password": "receiverPassword"}

	sendCoinSchema := map[string]interface{}{
		"toUser": "receiver",
		"amount": 10000}

	authBody, err := json.Marshal(senderUser)
	require.NoError(t, err)

	senderToken := authUser(t, authBody, http.StatusOK, true)

	authBody, err = json.Marshal(receiverUser)
	require.NoError(t, err)

	_ = authUser(t, authBody, http.StatusOK, true)

	sendBody, err := json.Marshal(sendCoinSchema)
	require.NoError(t, err)

	sendCoins(t, sendBody, senderToken.SignedToken, http.StatusBadRequest)
}

func TestSendNegativeAmount(t *testing.T) {
	senderUser := map[string]interface{}{
		"username": "sender",
		"password": "senderPassword"}

	receiverUser := map[string]interface{}{
		"username": "receiver",
		"password": "receiverPassword"}

	authBody, err := json.Marshal(senderUser)
	require.NoError(t, err)

	senderToken := authUser(t, authBody, http.StatusOK, true)

	authBody, err = json.Marshal(receiverUser)
	require.NoError(t, err)

	_ = authUser(t, authBody, http.StatusOK, true)

	sendCoinSchema := map[string]interface{}{
		"toUser": "receiver",
		"amount": -90}

	sendBody, err := json.Marshal(sendCoinSchema)
	require.NoError(t, err)

	sendCoins(t, sendBody, senderToken.SignedToken, http.StatusBadRequest)
}

func TestCorrectTransaction(t *testing.T) {
	firstUsername := "firstUser" + strconv.Itoa(rand.Int())
	secondUsername := "secondUser" + strconv.Itoa(rand.Int())

	firstUser := map[string]interface{}{
		"username": firstUsername,
		"password": "firstUserPassword"}

	secondUser := map[string]interface{}{
		"username": secondUsername,
		"password": "secondUserPassword"}

	authBody, err := json.Marshal(firstUser)
	require.NoError(t, err)
	firstUserToken := authUser(t, authBody, http.StatusOK, true)

	authBody, err = json.Marshal(secondUser)
	require.NoError(t, err)

	secondUserToken := authUser(t, authBody, http.StatusOK, true)

	//Первый отправляет второму 100 и 500 монет
	sendSchema := map[string]interface{}{
		"toUser": secondUsername,
		"amount": 100,
	}
	sendBody, err := json.Marshal(sendSchema)
	require.NoError(t, err)

	sendCoins(t, sendBody, firstUserToken.SignedToken, http.StatusOK)

	sendSchema = map[string]interface{}{
		"toUser": secondUsername,
		"amount": 500,
	}
	sendBody, err = json.Marshal(sendSchema)
	require.NoError(t, err)
	sendCoins(t, sendBody, firstUserToken.SignedToken, http.StatusOK)

	// Второй отправляет первому 400 монет

	sendSchema = map[string]interface{}{
		"toUser": firstUsername,
		"amount": 400,
	}
	sendBody, err = json.Marshal(sendSchema)
	require.NoError(t, err)
	sendCoins(t, sendBody, secondUserToken.SignedToken, http.StatusOK)

	// проверка истории переводов

	var targetFirstUserInfo = controllers.InfoSchema{
		Coins: 1000 - 100 - 500 + 400, // перевел 100, 500 второму, получил 400 от второго
		CoinHistory: controllers.HistorySchema{Received: []controllers.ReceivedSchema{
			{FromUser: secondUsername, Amount: 400},
		},
			Sent: []controllers.SentSchema{
				{ToUser: secondUsername, Amount: 100}, {ToUser: secondUsername, Amount: 500},
			}},
	}
	var targetSecondUserInfo = controllers.InfoSchema{
		Coins: 1000 + 100 + 500 - 400, // получил 100, 500 от первого, перевел 400 первому
		CoinHistory: controllers.HistorySchema{Received: []controllers.ReceivedSchema{
			{FromUser: firstUsername, Amount: 100}, {FromUser: firstUsername, Amount: 500},
		},
			Sent: []controllers.SentSchema{
				{ToUser: firstUsername, Amount: 400},
			}},
	}
	firstUserHistory := getInfo(t, firstUserToken.SignedToken, http.StatusOK, true)

	assert.Equal(t, targetFirstUserInfo.Coins, firstUserHistory.Coins)
	assert.Equal(t, targetFirstUserInfo.Inventory, firstUserHistory.Inventory)
	assert.Equal(t, targetFirstUserInfo.CoinHistory, firstUserHistory.CoinHistory)

	secondUserHistory := getInfo(t, secondUserToken.SignedToken, http.StatusOK, true)

	assert.Equal(t, targetSecondUserInfo.Coins, secondUserHistory.Coins)
	assert.Equal(t, targetSecondUserInfo.Inventory, secondUserHistory.Inventory)
	assert.Equal(t, targetSecondUserInfo.CoinHistory, secondUserHistory.CoinHistory)

}
