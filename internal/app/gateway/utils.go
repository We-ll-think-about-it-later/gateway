package gateway

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/golang-jwt/jwt"
)

func safeProxyRequest(secretKey []byte) func(w http.ResponseWriter, r *http.Request, upstream *url.URL) error {
	return func(w http.ResponseWriter, r *http.Request, upstream *url.URL) error {
		// Извлекаем токен из заголовка Authorization
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			return errors.New("Отсутствует токен авторизации")
		}

		// Проверяем токен
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Проверка что метод подпись токена ожидаем
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("Неожиданный метод подписи токена")
			}
			return secretKey, nil
		})

		if err != nil {
			return err
		}

		// Проверяем, действителен ли токен
		if _, ok := token.Claims.(jwt.MapClaims); !ok || !token.Valid {
			return errors.New("Недействительный токен")
		}

		return proxyRequest(w, r, upstream)
	}
}

func proxyRequest(w http.ResponseWriter, r *http.Request, upstream *url.URL) error {
	// Копируем запрос для отправки к upstream сервису
	proxyReq, err := http.NewRequest(r.Method, upstream.String(), r.Body)
	if err != nil {
		return err
	}

	// Копируем заголовки
	for header, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(header, value)
		}
	}

	// Отправляем запрос к upstream
	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Прокидываем ответ обратно клиенту
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func setCookie(w http.ResponseWriter, name, value string, maxAge int) error {
	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		MaxAge:   maxAge,
		HttpOnly: true,
		// Secure:   true, // Только https
		Path: "/",
	}
	http.SetCookie(w, cookie)
	return nil
}

func writeResponseBody(w http.ResponseWriter, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(body)
}

func forwardUpstreamResponse(w http.ResponseWriter, r *http.Request) {
	// Отправляем код статуса и тело ответа от upstream
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Ошибка чтения ответа", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(r.Response.StatusCode)
	w.Write(body)
}
