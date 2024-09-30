package gateway

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
)

var (
	ErrCantReadResponseBody       = errors.New("Ошибка чтения тела ответа")
	ErrCantUnmarshallResponseBody = errors.New("Ошибка разбора тела ответа")
	ErrMissingAccessToken         = errors.New("Токен доступа не найден в ответе")
	ErrMissingRefreshToken        = errors.New("Токен обновления не найден в ответе")
	ErrCantSetCookies             = errors.New("Не получилось установить куки")
)

var (
	refreshTokenCookieLifeTime = 2592000
)

func authTokenHandler(w http.ResponseWriter, r *http.Request, upstream *url.URL) error {
	err := proxyRequest(w, r, upstream)
	if err != nil {
		return err
	}

	if r.Response.StatusCode == http.StatusOK {
		body, err := io.ReadAll(r.Response.Body)
		if err != nil {
			return ErrCantReadResponseBody
		}

		var responseBody map[string]interface{}
		if err := json.Unmarshal(body, &responseBody); err != nil {
			return ErrCantUnmarshallResponseBody
		}

		_, ok := responseBody["access_token"].(string)
		if !ok {
			return ErrMissingAccessToken
		}

		refreshToken, ok := responseBody["refresh_token"].(string)
		if !ok {
			http.Error(w, "", http.StatusInternalServerError)
			return ErrMissingRefreshToken
		}

		// Убираем refresh токен из тела запроса, чтобы у клиента не было к нему прямого доступа
		delete(responseBody, "refresh_token")

		// Ставим куки с refresh токеном
		if err := setCookie(w, "refresh_token", refreshToken, refreshTokenCookieLifeTime); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return ErrCantSetCookies
		}

		writeResponseBody(w, responseBody)
	} else {
		forwardUpstreamResponse(w, r)
	}
	return nil
}
