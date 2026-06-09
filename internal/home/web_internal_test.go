package home

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestH2CPriorKnowledgeHandler_readHeaderTimeout(t *testing.T) {
	t.Parallel()

	const timeout = 50 * time.Millisecond

	srv := &http.Server{
		Handler: newH2CPriorKnowledgeHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})),
		ReadTimeout:       timeout,
		ReadHeaderTimeout: timeout,
	}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(l) }()

	t.Cleanup(func() {
		ctx := testutil.ContextWithTimeout(t, testTimeout)
		require.NoError(t, srv.Shutdown(ctx))
		require.ErrorIs(t, <-serveErr, http.ErrServerClosed)
	})

	conn, err := net.Dial("tcp", l.Addr().String())
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, conn.Close()) })

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(testTimeout)))

	_, err = conn.Read([]byte{0})
	require.Error(t, err)

	var netErr net.Error
	require.False(t, errors.As(err, &netErr) && netErr.Timeout())
}

func TestH2CPriorKnowledgeHandler_rejectsUpgrade(t *testing.T) {
	t.Parallel()

	hdlr := newH2CPriorKnowledgeHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	req.Header.Set("Connection", "upgrade")
	req.Header.Set("Upgrade", "h2c")

	w := httptest.NewRecorder()
	hdlr.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestH2CPriorKnowledgeHandler_passesHTTP1(t *testing.T) {
	t.Parallel()

	hdlr := newH2CPriorKnowledgeHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	w := httptest.NewRecorder()
	hdlr.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "http://example.test/", nil))

	assert.Equal(t, http.StatusNoContent, w.Code)
}
