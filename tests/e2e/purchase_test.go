package e2e

import (
	"avito/controllers"
	"encoding/json"
	"github.com/go-playground/assert/v2"
	"github.com/stretchr/testify/require"
	"math/rand"
	"net/http"
	"strconv"
	"testing"
	"time"
)

func buyItem(t *testing.T, item, token string, targetStatus int) {
	const buyItemUrl = "http://localhost:8080/api/buy/"

	req, err := http.NewRequest(http.MethodGet, buyItemUrl+item, nil)
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

func TestUnauthorizedPurchase(t *testing.T) {
	invalidToken := "not the actual token"
	item := "pen"

	buyItem(t, item, invalidToken, http.StatusUnauthorized)
}

func TestNoSuchItem(t *testing.T) {
	newUser := map[string]string{
		"username": "newUser",
		"password": "password"}
	authBody, err := json.Marshal(newUser)
	require.NoError(t, err)
	validToken := authUser(t, authBody, http.StatusOK, true)

	item := "nosuchitem"

	buyItem(t, item, validToken.SignedToken, http.StatusBadRequest)
}

func TestNotEnoughFunds(t *testing.T) {
	newUser := map[string]string{
		"username": "UserWith1000Coins" + strconv.Itoa(rand.Int()),
		"password": "password"}
	authBody, err := json.Marshal(newUser)
	require.NoError(t, err)
	validToken := authUser(t, authBody, http.StatusOK, true)

	item := "hoody"
	buyItem(t, item, validToken.SignedToken, http.StatusOK)
	buyItem(t, item, validToken.SignedToken, http.StatusOK)
	buyItem(t, item, validToken.SignedToken, http.StatusOK)
	buyItem(t, item, validToken.SignedToken, http.StatusBadRequest) // закончились деньги
}

func TestBuyItems(t *testing.T) {
	someUser := map[string]string{
		"username": "user" + strconv.Itoa(rand.Int()),
		"password": "internPassword"}
	authBody, err := json.Marshal(someUser)
	require.NoError(t, err)

	validToken := authUser(t, authBody, http.StatusOK, true)

	item := "pen"

	buyItem(t, item, validToken.SignedToken, http.StatusOK)
	buyItem(t, item, validToken.SignedToken, http.StatusOK)
	buyItem(t, item, validToken.SignedToken, http.StatusOK)

	item = "hoody"
	buyItem(t, item, validToken.SignedToken, http.StatusOK)
	buyItem(t, item, validToken.SignedToken, http.StatusOK)

	item = "t-shirt"
	buyItem(t, item, validToken.SignedToken, http.StatusOK)

	var targetInfoResp = controllers.InfoSchema{
		Coins: 1000 - 10*3 - 300*2 - 1*80, // стоимость pen=10, hoody=300, t-shirt=80
		Inventory: []controllers.InventorySchema{{Type: "pen", Quantity: 3},
			{Type: "hoody", Quantity: 2}, {Type: "t-shirt", Quantity: 1}},
	}
	history := getInfo(t, validToken.SignedToken, http.StatusOK, true)
	sortInventory(targetInfoResp.Inventory)
	sortInventory(history.Inventory)

	assert.Equal(t, targetInfoResp.Coins, history.Coins)
	assert.Equal(t, targetInfoResp.Inventory, history.Inventory)
	assert.Equal(t, targetInfoResp.CoinHistory, history.CoinHistory)

}
