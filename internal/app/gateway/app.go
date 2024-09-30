package gateway

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gateway/config"
	"gateway/internal/pkg/balancer"

	"github.com/gorilla/mux"
)

type UpstreamServices struct {
	IdentityService *balancer.Balancer
}

// Run запускает HTTP сервер и обрабатывает сигналы для корректного завершения работы.
func Run(cfg config.Config) {
	upstreamServices, err := newUpstreamServices(cfg.IdentityServiceAddresses)
	if err != nil {
		log.Fatalf("Ошибка разбора URL upstream сервисов: %v", err)
	}

	router := mux.NewRouter()
	safeProxyRequestWithSecret := safeProxyRequest([]byte(cfg.Secret))

	// Маршрутизация запросов
	router.PathPrefix("/auth/token").HandlerFunc(wrapHandler(upstreamServices.IdentityService, authTokenHandler)).Methods("POST")
	router.PathPrefix("/auth/").HandlerFunc(wrapHandler(upstreamServices.IdentityService, proxyRequest))
	// Во всех остальных ручках проверяется JWT
	router.PathPrefix("/users/").HandlerFunc(wrapHandler(upstreamServices.IdentityService, safeProxyRequestWithSecret))

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler: router,
	}

	// Запуск сервера в отдельной горутине
	go func() {
		log.Printf("Запуск сервера на порту %d", cfg.HTTP.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка запуска сервера: %s", err)
		}
	}()

	// Ожидание сигналов завершения работы
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Завершение работы сервера...")

	// Контекст для завершения сервера с таймаутом 5 секунд
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Принудительное завершение работы сервера: %s", err)
	}

	log.Println("Сервер завершил работу")
}

// wrapHandler оборачивает хендлер с логикой балансировки.
func wrapHandler(balancer *balancer.Balancer, handler func(http.ResponseWriter, *http.Request, *url.URL) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		upstream := balancer.Next()
		if upstream == nil {
			http.Error(w, "Нет доступных upstream сервисов", http.StatusServiceUnavailable)
			return
		}
		handler(w, r, upstream)
	}
}
