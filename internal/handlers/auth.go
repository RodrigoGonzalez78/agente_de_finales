package handlers

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct{}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

func (h *AuthHandler) ShowLogin(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", nil)
}

func (h *AuthHandler) Login(c *gin.Context) {
	email := c.PostForm("email")
	password := c.PostForm("password")

	envEmail := os.Getenv("ADMIN_EMAIL")
	envPass := os.Getenv("ADMIN_PASSWORD")

	if email == envEmail && password == envPass {
		// Set cookie
		c.SetCookie("admin_session", "logged_in", 3600, "/", "", false, true)
		c.Redirect(http.StatusFound, "/admin")
	} else {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "Credenciales incorrectas"})
	}
}

func (h *AuthHandler) Logout(c *gin.Context) {
	c.SetCookie("admin_session", "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/login")
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie("admin_session")
		if err != nil || cookie != "logged_in" {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		c.Next()
	}
}
