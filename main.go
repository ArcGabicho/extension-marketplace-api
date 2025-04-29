package main

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	"extensions-api/data"
)

// Configuración de Rate Limiting
var (
	clients          = make(map[string]*rate.Limiter) // Mapa de clientes por IP
	requestsPerMinute = 10                            // Límite de solicitudes
	burst            = 5                              // Picos permitidos
)

// Middleware de Rate Limiting
func rateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		// Inicializar limiter para IPs nuevas
		if _, exists := clients[ip]; !exists {
			clients[ip] = rate.NewLimiter(rate.Every(time.Minute/time.Duration(requestsPerMinute)), burst)
		}

		// Rechazar si excede el límite
		if !clients[ip].Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Máximo 10 solicitudes por minuto",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Middleware de Seguridad Adicional
func securityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Bloquear User-Agents sospechosos
		userAgent := c.GetHeader("User-Agent")
		if strings.Contains(userAgent, "Python-urllib") || strings.Contains(userAgent, "curl") {
			c.JSON(http.StatusForbidden, gin.H{"error": "automated_access_denied"})
			c.Abort()
			return
		}

		// 2. CORS Estricto (permite solo tu frontend)
		c.Header("Access-Control-Allow-Origin", "https://localhost:4321/") // Cambia esto
		c.Header("Access-Control-Allow-Methods", "GET")

		c.Next()
	}
}

func main() {
	r := gin.Default()

	// Middlewares globales
	r.Use(securityMiddleware())
	r.Use(rateLimitMiddleware())

	// Endpoints
	r.GET("/api/extensions", getExtensions)
	r.GET("/api/extensions/:id", getExtensionByID)
	r.GET("/api/extensions/search/:query", searchExtensions)

	// Iniciar servidor
	log.Println("🚀 Servidor iniciado en http://localhost:8080")
	r.Run(":8080")
}

// Handlers (sin cambios, solo para referencia)
func getExtensions(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=300") // Cache de 5 minutos
	c.JSON(http.StatusOK, gin.H{
		"count":      len(data.GetExtensions()),
		"extensions": data.GetExtensions(),
	})
}

func getExtensionByID(c *gin.Context) {
	id := c.Param("id")
	for _, ext := range data.GetExtensions() {
		if ext.ID == id {
			c.JSON(http.StatusOK, ext)
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "extension_not_found"})
}

func searchExtensions(c *gin.Context) {
	query := c.Param("query")
	var results []data.Extension

	for _, ext := range data.GetExtensions() {
		if strings.Contains(strings.ToLower(ext.Name), strings.ToLower(query)) {
			results = append(results, ext)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"count":      len(results),
		"extensions": results,
	})
}