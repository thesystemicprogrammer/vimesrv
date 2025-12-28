package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORSMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "ok"})
	})

	t.Run("adds CORS headers", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Errorf("Access-Control-Allow-Origin = %s, want *", w.Header().Get("Access-Control-Allow-Origin"))
		}

		if w.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Error("Access-Control-Allow-Methods not set")
		}
	})

	t.Run("handles OPTIONS request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("OPTIONS", "/test", nil)
		router.ServeHTTP(w, req)

		if w.Code != 204 {
			t.Errorf("status code = %d, want 204", w.Code)
		}
	})
}

func TestErrorHandlerMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("does not affect successful requests", func(t *testing.T) {
		router := gin.New()
		router.Use(ErrorHandlerMiddleware())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "ok"})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("status code = %d, want 200", w.Code)
		}
	})
}

func TestRecoveryMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("recovers from panic", func(t *testing.T) {
		router := gin.New()
		router.Use(RecoveryMiddleware())
		router.GET("/test", func(c *gin.Context) {
			panic("test panic")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		if w.Code != 500 {
			t.Errorf("status code = %d, want 500", w.Code)
		}
	})

	t.Run("does not affect normal requests", func(t *testing.T) {
		router := gin.New()
		router.Use(RecoveryMiddleware())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "ok"})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("status code = %d, want 200", w.Code)
		}
	})
}

func TestLoggingMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("logs requests without errors", func(t *testing.T) {
		router := gin.New()
		router.Use(LoggingMiddleware())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "ok"})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("status code = %d, want 200", w.Code)
		}
	})

	t.Run("logs requests with query parameters", func(t *testing.T) {
		router := gin.New()
		router.Use(LoggingMiddleware())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "ok"})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test?param=value", nil)
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("status code = %d, want 200", w.Code)
		}
	})
}
