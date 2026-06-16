package server

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"

	"docklog/access"
	"docklog/config"
	"docklog/db"
	"docklog/k8s"
	"docklog/middleware"
	"docklog/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

func (s *Server) registerK8sLogRoutes(r *echo.Group) {
	if s.deps.K8s == nil {
		return
	}

	r.GET("/namespaces/:namespace/pods/:pod/logs", s.handlePodLogs)
	r.GET("/namespaces/:namespace/pods/:pod/logs/download", s.handlePodLogsDownload)
	r.GET("/namespaces/:namespace/pods/:pod/logs/count", s.handlePodLogsCount)
}

func (s *Server) registerK8sWebSocketRoutes() {
	if s.deps.K8s == nil {
		return
	}

	s.echo.GET("/ws/pod-logs/:namespace/:pod", s.handlePodLogsWebSocket)
	s.echo.GET("/ws/pod-shell/:namespace/:pod", s.handlePodShellWebSocket)
}

func (s *Server) authorizePodAccess(userClaims *models.UserClaims, namespace, podName string) error {
	if !config.K8sNamespaceAllowed(namespace) {
		return echo.NewHTTPError(http.StatusForbidden, map[string]string{"error": "namespace is not allowed"})
	}

	var dbIsAdmin bool
	db.DB.QueryRow("SELECT COALESCE(is_admin, 0) FROM users WHERE id = ?", userClaims.ID).Scan(&dbIsAdmin)
	if dbIsAdmin {
		return nil
	}

	patterns := access.GetAuthorizedPatterns(userClaims.ID)
	if !access.NamespaceVisible(patterns, namespace) {
		return echo.NewHTTPError(http.StatusForbidden, map[string]string{"error": "Access Denied"})
	}
	if access.PodAllowed(patterns, namespace, podName) {
		return nil
	}
	return echo.NewHTTPError(http.StatusForbidden, map[string]string{"error": "Access Denied"})
}

func (s *Server) handlePodLogs(c echo.Context) error {
	namespace := strings.TrimSpace(c.Param("namespace"))
	podName := strings.TrimSpace(c.Param("pod"))
	token := c.Get("user").(*jwt.Token)
	userClaims := token.Claims.(*models.UserClaims)

	if err := s.authorizePodAccess(userClaims, namespace, podName); err != nil {
		return err
	}

	tail := 100
	if tailStr := c.QueryParam("tail"); tailStr != "" {
		if tailVal, err := strconv.Atoi(tailStr); err == nil && tailVal > 0 {
			tail = tailVal
		}
	}

	logs, err := k8s.ReadPodLogs(context.Background(), s.deps.K8s, namespace, podName, k8s.PodLogRequest{
		Container:  c.QueryParam("container"),
		Tail:       tail,
		Until:      c.QueryParam("until"),
		Since:      c.QueryParam("since"),
		Timestamps: true,
	})
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, logs)
}

func (s *Server) handlePodLogsDownload(c echo.Context) error {
	namespace := strings.TrimSpace(c.Param("namespace"))
	podName := strings.TrimSpace(c.Param("pod"))
	token := c.Get("user").(*jwt.Token)
	userClaims := token.Claims.(*models.UserClaims)

	if err := s.authorizePodAccess(userClaims, namespace, podName); err != nil {
		return err
	}

	sinceStr := c.QueryParam("since")
	untilStr := c.QueryParam("until")
	sinceTime, err := parseLogTimeQuery(sinceStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid since timestamp"})
	}
	untilTime, err := parseLogTimeQuery(untilStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid until timestamp"})
	}
	if !sinceTime.IsZero() && !untilTime.IsZero() && sinceTime.After(untilTime) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "since must be before until"})
	}

	lines, err := k8s.ReadPodLogs(context.Background(), s.deps.K8s, namespace, podName, k8s.PodLogRequest{
		Container:  c.QueryParam("container"),
		Tail:       100000,
		Since:      sinceStr,
		Until:      untilStr,
		Timestamps: true,
	})
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
	}

	filtered := filterLogLinesByTime(lines, sinceTime, untilTime)
	body := strings.Join(filtered, "\n")
	if body != "" {
		body += "\n"
	}

	s.audit(userClaims.ID, userClaims.Username, "DOWNLOAD_POD_LOGS", namespace+"/"+podName, "Success", "Pod log archive exported")

	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+podName+"_"+namespace+".log")
	c.Response().Header().Set(echo.HeaderContentType, "text/plain")
	return c.String(http.StatusOK, body)
}

func (s *Server) handlePodLogsCount(c echo.Context) error {
	namespace := strings.TrimSpace(c.Param("namespace"))
	podName := strings.TrimSpace(c.Param("pod"))
	token := c.Get("user").(*jwt.Token)
	userClaims := token.Claims.(*models.UserClaims)

	if err := s.authorizePodAccess(userClaims, namespace, podName); err != nil {
		return err
	}

	count, err := k8s.CountPodLogs(context.Background(), s.deps.K8s, namespace, podName, c.QueryParam("container"))
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]int{"total": count})
}

func (s *Server) handlePodLogsWebSocket(c echo.Context) error {
	namespace := strings.TrimSpace(c.Param("namespace"))
	podName := strings.TrimSpace(c.Param("pod"))

	userClaims, err := middleware.AuthenticateWS(c)
	if err != nil {
		return middleware.WSAuthError(c, err)
	}

	if err := s.authorizePodAccess(userClaims, namespace, podName); err != nil {
		if httpErr, ok := err.(*echo.HTTPError); ok {
			return c.JSON(httpErr.Code, httpErr.Message)
		}
		return err
	}

	ws, err := middleware.UpgradeAuthenticatedWS(c)
	if err != nil {
		return err
	}
	defer ws.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	container := c.QueryParam("container")
	stream, err := k8s.StreamPodLogs(ctx, s.deps.K8s, namespace, podName, container, 100)
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
	}
	defer stream.Close()

	scanner := bufio.NewScanner(stream)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if err := ws.WriteMessage(websocket.TextMessage, []byte(scanner.Text())); err != nil {
			break
		}
	}
	return nil
}

func (s *Server) handlePodShellWebSocket(c echo.Context) error {
	namespace := strings.TrimSpace(c.Param("namespace"))
	podName := strings.TrimSpace(c.Param("pod"))

	userClaims, err := middleware.AuthenticateWS(c)
	if err != nil {
		return middleware.WSAuthError(c, err)
	}
	if !config.AllowShell {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Shell access is disabled on this server."})
	}
	var canShell bool
	err = db.DB.QueryRow("SELECT can_shell FROM users WHERE id = ? AND is_active = 1", userClaims.ID).Scan(&canShell)
	if err != nil || !canShell {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Shell access is not permitted for this account."})
	}
	if err := s.authorizePodAccess(userClaims, namespace, podName); err != nil {
		if httpErr, ok := err.(*echo.HTTPError); ok {
			return c.JSON(httpErr.Code, httpErr.Message)
		}
		return err
	}

	ws, err := middleware.UpgradeAuthenticatedWS(c)
	if err != nil {
		return err
	}
	defer ws.Close()

	shellCmd := c.QueryParam("shell")
	if shellCmd == "" {
		shellCmd = "/bin/sh"
	}
	allowedShells := map[string]bool{"/bin/sh": true, "/bin/bash": true, "/bin/ash": true}
	if !allowedShells[shellCmd] {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid shell"})
	}

	container := strings.TrimSpace(c.QueryParam("container"))
	if container == "" {
		pod, getErr := s.deps.K8s.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
		if getErr != nil || len(pod.Spec.Containers) == 0 {
			_ = ws.WriteMessage(websocket.TextMessage, []byte("\r\n[DockLog] Unable to determine pod container.\r\n"))
			return nil
		}
		container = pod.Spec.Containers[0].Name
	}

	restCfg, err := k8s.NewRESTConfig()
	if err != nil {
		_ = ws.WriteMessage(websocket.TextMessage, []byte("\r\n[DockLog] Kubernetes config unavailable: "+err.Error()+"\r\n"))
		return nil
	}

	req := s.deps.K8s.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec")
	req.VersionedParams(&corev1.PodExecOptions{
		Container: container,
		Command:   []string{shellCmd},
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(restCfg, http.MethodPost, req.URL())
	if err != nil {
		_ = ws.WriteMessage(websocket.TextMessage, []byte("\r\n[DockLog] Failed to start pod shell: "+err.Error()+"\r\n"))
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	inReader := newWSInputReader()
	outWriter := &wsOutputWriter{ws: ws}
	errChan := make(chan error, 2)

	go func() {
		defer inReader.Close()
		for {
			_, msg, readErr := ws.ReadMessage()
			if readErr != nil {
				errChan <- readErr
				return
			}
			inReader.Push(msg)
		}
	}()

	go func() {
		streamErr := executor.StreamWithContext(ctx, remotecommand.StreamOptions{
			Stdin:  inReader,
			Stdout: outWriter,
			Stderr: outWriter,
			Tty:    true,
		})
		errChan <- streamErr
	}()

	<-errChan
	return nil
}

type wsInputReader struct {
	ch  chan []byte
	buf []byte
}

func newWSInputReader() *wsInputReader {
	return &wsInputReader{ch: make(chan []byte, 16)}
}

func (r *wsInputReader) Push(data []byte) {
	if len(data) == 0 {
		return
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	r.ch <- cp
}

func (r *wsInputReader) Close() {
	close(r.ch)
}

func (r *wsInputReader) Read(p []byte) (int, error) {
	for len(r.buf) == 0 {
		data, ok := <-r.ch
		if !ok {
			return 0, io.EOF
		}
		r.buf = data
	}
	n := copy(p, r.buf)
	r.buf = r.buf[n:]
	return n, nil
}

type wsOutputWriter struct {
	ws *websocket.Conn
}

func (w *wsOutputWriter) Write(p []byte) (int, error) {
	if err := w.ws.WriteMessage(websocket.TextMessage, p); err != nil {
		return 0, err
	}
	return len(p), nil
}
