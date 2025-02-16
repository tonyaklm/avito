package e2e

import (
	"avito/controllers"
	"encoding/json"
	"github.com/go-playground/assert/v2"
	"github.com/stretchr/testify/require"
	"io"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"testing"
	"time"
)

func getInfo(t *testing.T, token string, targetStatus int, parseBody bool) controllers.InfoSchema {
	const infoUrl = "http://localhost:8080/api/info"

	req, err := http.NewRequest(http.MethodGet, infoUrl, nil)
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
	if !parseBody {
		return controllers.InfoSchema{}
	}
	var info controllers.InfoSchema
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	err = json.Unmarshal(body, &info)
	require.NoError(t, err)
	return info
}

func sortInventory(in []controllers.InventorySchema) {
	sort.Slice(in, func(i, j int) bool {
		if in[i].Quantity != in[j].Quantity {
			return in[i].Quantity < in[j].Quantity
		}
		return in[i].Type < in[j].Type
	})
}

func TestUnauthorizedInfo(t *testing.T) {
	invalidToken := "not the actual token"

	getInfo(t, invalidToken, http.StatusUnauthorized, false)
}

func TestGetInfo(t *testing.T) {
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

	// первый покупает 3 book=50, 2 powerbank=200

	item := "book"

	buyItem(t, item, firstUserToken.SignedToken, http.StatusOK)
	buyItem(t, item, firstUserToken.SignedToken, http.StatusOK)
	buyItem(t, item, firstUserToken.SignedToken, http.StatusOK)

	item = "powerbank"
	buyItem(t, item, firstUserToken.SignedToken, http.StatusOK)
	buyItem(t, item, firstUserToken.SignedToken, http.StatusOK)

	// второй покупает 1 pink-hoody=500, 1 cup=20, 2 t-shirt=80

	item = "pink-hoody"
	buyItem(t, item, secondUserToken.SignedToken, http.StatusOK)

	item = "cup"
	buyItem(t, item, secondUserToken.SignedToken, http.StatusOK)

	item = "t-shirt"
	buyItem(t, item, secondUserToken.SignedToken, http.StatusOK)
	buyItem(t, item, secondUserToken.SignedToken, http.StatusOK)

	//Превый отправляет второму по 40, 300 монет

	sendSchema := map[string]interface{}{
		"toUser": secondUsername,
		"amount": 40,
	}
	sendBody, err := json.Marshal(sendSchema)
	require.NoError(t, err)

	sendCoins(t, sendBody, firstUserToken.SignedToken, http.StatusOK)

	sendSchema["amount"] = 300
	sendBody, err = json.Marshal(sendSchema)
	require.NoError(t, err)
	sendCoins(t, sendBody, firstUserToken.SignedToken, http.StatusOK)

	//Второй отправляет первому 100, 90
	sendSchema = map[string]interface{}{
		"toUser": firstUsername,
		"amount": 100,
	}
	sendBody, err = json.Marshal(sendSchema)
	require.NoError(t, err)

	sendCoins(t, sendBody, secondUserToken.SignedToken, http.StatusOK)

	sendSchema["amount"] = 90
	sendBody, err = json.Marshal(sendSchema)
	require.NoError(t, err)
	sendCoins(t, sendBody, secondUserToken.SignedToken, http.StatusOK)

	// проверка истории переводов и товаров для первого:  3 book=50, 2 powerbank=200

	var targetFirstUserInfo = controllers.InfoSchema{
		Coins: 1000 - 3*50 - 2*200 - 40 - 300 + 90 + 100,
		CoinHistory: controllers.HistorySchema{Received: []controllers.ReceivedSchema{
			{FromUser: secondUsername, Amount: 100}, {FromUser: secondUsername, Amount: 90},
		},
			Sent: []controllers.SentSchema{
				{ToUser: secondUsername, Amount: 40}, {ToUser: secondUsername, Amount: 300},
			},
		},
		Inventory: []controllers.InventorySchema{{Type: "book", Quantity: 3},
			{Type: "powerbank", Quantity: 2}},
	}

	firstUserHistory := getInfo(t, firstUserToken.SignedToken, http.StatusOK, true)
	sortInventory(targetFirstUserInfo.Inventory)
	sortInventory(firstUserHistory.Inventory)

	assert.Equal(t, targetFirstUserInfo.Coins, firstUserHistory.Coins)
	assert.Equal(t, targetFirstUserInfo.Inventory, firstUserHistory.Inventory)
	assert.Equal(t, targetFirstUserInfo.CoinHistory, firstUserHistory.CoinHistory)

	// проверка истории переводов и товаров для второго: 1 pink-hoody=500, 1 cup=20, 2 t-shirt=80

	var targetSecondUserInfo = controllers.InfoSchema{
		Coins: 1000 - 1*500 - 1*20 - 2*80 + 40 + 300 - 90 - 100,
		CoinHistory: controllers.HistorySchema{Received: []controllers.ReceivedSchema{
			{FromUser: firstUsername, Amount: 40}, {FromUser: firstUsername, Amount: 300},
		},
			Sent: []controllers.SentSchema{
				{ToUser: firstUsername, Amount: 100}, {ToUser: firstUsername, Amount: 90},
			},
		},
		Inventory: []controllers.InventorySchema{{Type: "pink-hoody", Quantity: 1},
			{Type: "cup", Quantity: 1}, {Type: "t-shirt", Quantity: 2}},
	}

	secondUserHistory := getInfo(t, secondUserToken.SignedToken, http.StatusOK, true)
	sortInventory(targetSecondUserInfo.Inventory)
	sortInventory(secondUserHistory.Inventory)

	assert.Equal(t, targetSecondUserInfo.Coins, secondUserHistory.Coins)
	assert.Equal(t, targetSecondUserInfo.Inventory, secondUserHistory.Inventory)
	assert.Equal(t, targetSecondUserInfo.CoinHistory, secondUserHistory.CoinHistory)

}
