package e2e

import (
	"avito/controllers"
	"bytes"
	"encoding/json"
	"github.com/go-playground/assert/v2"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"testing"
	"time"
)

func authUser(t *testing.T, authBody []byte, targetStatus int, parseBody bool) controllers.TokenResponse {

	const authUrl = "http://localhost:8080/api/auth"

	req, err := http.NewRequest(http.MethodPost, authUrl, bytes.NewBuffer(authBody))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("accept", "application/json")

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	res, err := client.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, targetStatus, res.StatusCode)
	if !parseBody {
		return controllers.TokenResponse{}
	}
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	var tokenBody controllers.TokenResponse
	err = json.Unmarshal(body, &tokenBody)
	require.NoError(t, err)
	return tokenBody
}

func TestCreateIncorrectUser(t *testing.T) {
	incorrectUser := map[string]interface{}{
		"username": 12,
		"password": "password"}
	authBody, err := json.Marshal(incorrectUser)
	require.NoError(t, err)

	authUser(t, authBody, http.StatusBadRequest, false)
}

func TestCreateUser(t *testing.T) {
	correctUser := map[string]string{
		"username": "user",
		"password": "password"}

	authBody, err := json.Marshal(correctUser)
	require.NoError(t, err)

	authUser(t, authBody, http.StatusOK, true)
}

func TestUnauthorizedUser(t *testing.T) {
	incorrectPasswordUser := map[string]string{
		"username": "user",
		"password": "incorrectPassword"}

	authBody, err := json.Marshal(incorrectPasswordUser)
	require.NoError(t, err)

	authUser(t, authBody, http.StatusUnauthorized, false)
}

func TestAuth(t *testing.T) {
	TestCreateUser(t)
}
