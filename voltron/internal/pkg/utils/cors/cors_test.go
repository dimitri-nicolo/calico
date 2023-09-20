package cors

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCORS(t *testing.T) {
	for _, tc := range []struct {
		name       string
		request    func() *http.Request
		nextCalled bool
		assertions func(t *testing.T, header http.Header)
	}{
		{
			name:       "no cors headers set if origin does not match",
			request:    func() *http.Request { return httptest.NewRequest(http.MethodGet, "/any", nil) },
			nextCalled: true,
			assertions: func(t *testing.T, header http.Header) {
				require.Empty(t, header.Get("Access-Control-Allow-Origin"))
				require.Empty(t, header.Get("Vary"))
				require.Empty(t, header.Get("Access-Control-Allow-Credentials"))
				require.Empty(t, header.Get("Access-Control-Allow-Methods"))
				require.Empty(t, header.Get("Access-Control-Allow-Headers"))
			},
		},
		{
			name: "cors set if origin matches",
			request: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/any", nil)
				req.Header.Set("origin", "https://calicocloud.io")
				return req
			},
			nextCalled: true,
			assertions: func(t *testing.T, header http.Header) {
				require.Equal(t, "https://calicocloud.io", header.Get("Access-Control-Allow-Origin"))
				require.Equal(t, "Origin", header.Get("Vary"))

				// options headers not set since not an Options request
				require.Empty(t, header.Get("Access-Control-Allow-Methods"))
				require.Empty(t, header.Get("Access-Control-Allow-Headers"))
			},
		},
		{
			name: "cors options requests",
			request: func() *http.Request {
				req := httptest.NewRequest(http.MethodOptions, "/any", nil)
				req.Header.Set("origin", "https://calicocloud.io")
				return req
			},
			nextCalled: false,
			assertions: func(t *testing.T, header http.Header) {
				require.Equal(t, "https://calicocloud.io", header.Get("Access-Control-Allow-Origin"))
				require.Equal(t, "Origin", header.Get("Vary"))

				// additional headers set
				require.NotEmpty(t, header.Get("Access-Control-Allow-Methods"))
				require.NotEmpty(t, header.Get("Access-Control-Allow-Headers"))
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := New(`https://calicocloud\.io`)
			require.NoError(t, err)

			nextCalled := false
			h := c.NewHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
			})

			recorder := httptest.NewRecorder()
			h.ServeHTTP(recorder, tc.request())
			require.Equal(t, tc.nextCalled, nextCalled)
			tc.assertions(t, recorder.Result().Header)
		})
	}
}
