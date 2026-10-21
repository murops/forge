package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/murops/forge/backend/internal/requests"
)

type Service interface {
	Templates(ctx context.Context) ([]requests.Template, error)
	Create(ctx context.Context, in requests.CreateInput, u requests.User) (requests.Request, error)
	Get(ctx context.Context, id uuid.UUID, u requests.User) (requests.Request, error)
	List(ctx context.Context, u requests.User) ([]requests.Request, error)
}

type Pinger interface {
	Ping(ctx context.Context) error
}

type handler struct {
	svc Service
	log *slog.Logger
}

func NewRouter(svc Service, verifier TokenVerifier, db Pinger, log *slog.Logger) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), accessLog(log))

	r.GET("/healthz", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			abort(c, http.StatusServiceUnavailable, "database unavailable")
			return
		}
		c.Status(http.StatusOK)
	})

	h := handler{svc: svc, log: log}
	v1 := r.Group("/api/v1", requireAuth(verifier))
	v1.GET("/templates", h.listTemplates)
	v1.GET("/repositories", h.listRequests)
	v1.POST("/repositories", h.createRequest)
	v1.GET("/repositories/:id", h.getRequest)

	return r
}

func (h handler) listTemplates(c *gin.Context) {
	ts, err := h.svc.Templates(c.Request.Context())
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"templates": ts})
}

func (h handler) listRequests(c *gin.Context) {
	rs, err := h.svc.List(c.Request.Context(), currentUser(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"repositories": rs})
}

func (h handler) createRequest(c *gin.Context) {
	var in requests.CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		abort(c, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req, err := h.svc.Create(c.Request.Context(), in, currentUser(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	c.Header("Location", "/api/v1/repositories/"+req.ID.String())
	c.JSON(http.StatusAccepted, req)
}

func (h handler) getRequest(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		abort(c, http.StatusNotFound, "not found")
		return
	}
	req, err := h.svc.Get(c.Request.Context(), id, currentUser(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, req)
}

func (h handler) fail(c *gin.Context, err error) {
	var verr *requests.ValidationError
	switch {
	case errors.As(err, &verr):
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{"error": verr.Msg, "field": verr.Field})
	case errors.Is(err, requests.ErrNotFound):
		abort(c, http.StatusNotFound, "not found")
	case errors.Is(err, requests.ErrTemplateNotFound):
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error(), "field": "template"})
	case errors.Is(err, requests.ErrNameTaken):
		abort(c, http.StatusConflict, err.Error())
	default:
		h.log.Error("request failed", "path", c.FullPath(), "error", err)
		abort(c, http.StatusInternalServerError, "internal error")
	}
}

func abort(c *gin.Context, status int, msg string) {
	c.AbortWithStatusJSON(status, gin.H{"error": msg})
}

func accessLog(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		if c.FullPath() == "/healthz" || c.FullPath() == "/readyz" {
			return
		}
		log.Info("http request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds())
	}
}
