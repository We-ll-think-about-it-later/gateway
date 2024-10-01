package gateway

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gateway/config"
	"gateway/internal/pkg/balancer"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// UpstreamServices структура для хранения балансировщиков upstream сервисов.
type UpstreamServices struct {
	IdentityService *balancer.Balancer
}

// init настраивает logrus.
func init() {
	logrus.SetOutput(os.Stdout)
}

// Run запускает HTTP сервер и обрабатывает сигналы для корректного завершения работы.
func Run(cfg config.Config) {
	upstreamServices, err := newUpstreamServices(cfg.IdentityServiceAddresses)
	if err != nil {
		logrus.WithError(err).Fatal("Не удалось инициализировать upstream сервисы")
	}

	router := mux.NewRouter()
	safeProxyRequestWithSecret := safeProxyRequest([]byte(cfg.Secret))

	router.PathPrefix("/auth/token").HandlerFunc(wrapHandler(upstreamServices.IdentityService, authTokenHandler, "authTokenHandler")).Methods("POST")
	router.PathPrefix("/auth/").HandlerFunc(wrapHandler(upstreamServices.IdentityService, proxyRequest, "proxyRequest"))
	router.PathPrefix("/users/").HandlerFunc(wrapHandler(upstreamServices.IdentityService, safeProxyRequestWithSecret, "safeProxyRequestWithSecret"))

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler: router,
	}

	go func() {
		logrus.Infof("Запуск сервера на порту %d", cfg.HTTP.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Fatal("Ошибка запуска сервера")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logrus.Info("Завершение работы сервера...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logrus.WithError(err).Fatal("Принудительное завершение работы сервера")
	}

	logrus.Info("Сервер завершил работу")
}

// wrapHandler оборачивает хендлер с логикой балансировки и логированием.
func wrapHandler(balancer *balancer.Balancer, handler func(http.ResponseWriter, *http.Request, *url.URL) error, handlerName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		upstream := balancer.Next()
		if upstream == nil {
			logrus.Error("Нет доступных upstream сервисов")
			http.Error(w, "Нет доступных upstream сервисов", http.StatusServiceUnavailable)
			return
		}

		requestLogger := logrus.WithFields(logrus.Fields{
			"upstream":    upstream.String(),
			"method":      r.Method,
			"path":        r.URL.Path,
			"handlerName": handlerName,
		})

		requestLogger.Info("Перенаправление запроса на upstream")

		err := handler(w, r, upstream)
		if err != nil {
			requestLogger.WithError(err).Error("Ошибка обработки запроса")
		}
	}
}
