package auth

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"testing"

	"github.com/bluenviron/gortsplib/v4/pkg/auth"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/headers"
	"github.com/most-1995/mediamtx/internal/conf"
	"github.com/stretchr/testify/require"
)

func mustParseCIDR(v string) net.IPNet {
	_, ne, err := net.ParseCIDR(v)
	if err != nil {
		panic(err)
	}
	if ipv4 := ne.IP.To4(); ipv4 != nil {
		return net.IPNet{IP: ipv4, Mask: ne.Mask[len(ne.Mask)-4 : len(ne.Mask)]}
	}
	return *ne
}

type testHTTPAuthenticator struct {
	*http.Server
}

func (ts *testHTTPAuthenticator) initialize(t *testing.T, protocol string, action string) {
	firstReceived := false

	ts.Server = &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, "/auth", r.URL.Path)

			var in struct {
				IP       string `json:"ip"`
				User     string `json:"user"`
				Password string `json:"password"`
				Path     string `json:"path"`
				Protocol string `json:"protocol"`
				ID       string `json:"id"`
				Action   string `json:"action"`
				Query    string `json:"query"`
			}
			err := json.NewDecoder(r.Body).Decode(&in)
			require.NoError(t, err)

			var user string
			if action == "publish" {
				user = "testpublisher"
			} else {
				user = "testreader"
			}

			if in.IP != "127.0.0.1" ||
				in.User != user ||
				in.Password != "testpass" ||
				in.Path != "teststream" ||
				in.Protocol != protocol ||
				(firstReceived && in.ID == "") ||
				in.Action != action ||
				(in.Query != "user=testreader&pass=testpass&param=value" &&
					in.Query != "user=testpublisher&pass=testpass&param=value" &&
					in.Query != "param=value") {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			firstReceived = true
		}),
	}

	ln, err := net.Listen("tcp", "127.0.0.1:9120")
	require.NoError(t, err)

	go ts.Server.Serve(ln)
}

func (ts *testHTTPAuthenticator) close() {
	ts.Server.Shutdown(context.Background())
}

func TestAuthInternal(t *testing.T) {
	for _, outcome := range []string{
		"ok",
		"wrong user",
		"wrong pass",
		"wrong ip",
		"wrong action",
		"wrong path",
	} {
		for _, encryption := range []string{
			"plain",
			"sha256",
			"argon2",
		} {
			t.Run(outcome+" "+encryption, func(t *testing.T) {
				m := Manager{
					Method: conf.AuthMethodInternal,
					InternalUsers: []conf.AuthInternalUser{
						{
							IPs: conf.IPNetworks{mustParseCIDR("127.1.1.1/32")},
							Permissions: []conf.AuthInternalUserPermission{
								{
									Action: conf.AuthActionPublish,
									Path:   "mypath",
								},
							},
						},
					},
					HTTPAddress:     "",
					RTSPAuthMethods: nil,
				}

				switch encryption {
				case "plain":
					m.InternalUsers[0].User = conf.Credential("testuser")
					m.InternalUsers[0].Pass = conf.Credential("testpass")

				case "sha256":
					m.InternalUsers[0].User = conf.Credential("sha256:rl3rgi4NcZkpAEcacZnQ2VuOfJ0FxAqCRaKB/SwdZoQ=")
					m.InternalUsers[0].Pass = conf.Credential("sha256:E9JJ8stBJ7QM+nV4ZoUCeHk/gU3tPFh/5YieiJp6n2w=")

				case "argon2":
					m.InternalUsers[0].User = conf.Credential(
						"argon2:$argon2id$v=19$m=4096,t=3,p=1$MTIzNDU2Nzg$Ux/LWeTgJQPyfMMJo1myR64+o8rALHoPmlE1i/TR+58")
					m.InternalUsers[0].Pass = conf.Credential(
						"argon2:$argon2i$v=19$m=4096,t=3,p=1$MTIzNDU2Nzg$/mrZ42TiTv1mcPnpMUera5oi0SFYbbyueAbdx5sUvWo")
				}

				switch outcome {
				case "ok":
					err := m.Authenticate(&Request{
						User:   "testuser",
						Pass:   "testpass",
						IP:     net.ParseIP("127.1.1.1"),
						Action: conf.AuthActionPublish,
						Path:   "mypath",
					})
					require.NoError(t, err)

				case "wrong user":
					err := m.Authenticate(&Request{
						User:   "wrong",
						Pass:   "testpass",
						IP:     net.ParseIP("127.1.1.1"),
						Action: conf.AuthActionPublish,
						Path:   "mypath",
					})
					require.Error(t, err)

				case "wrong pass":
					err := m.Authenticate(&Request{
						User:   "testuser",
						Pass:   "wrong",
						IP:     net.ParseIP("127.1.1.1"),
						Action: conf.AuthActionPublish,
						Path:   "mypath",
					})
					require.Error(t, err)

				case "wrong ip":
					err := m.Authenticate(&Request{
						User:   "testuser",
						Pass:   "testpass",
						IP:     net.ParseIP("127.1.1.2"),
						Action: conf.AuthActionPublish,
						Path:   "mypath",
					})
					require.Error(t, err)

				case "wrong action":
					err := m.Authenticate(&Request{
						User:   "testuser",
						Pass:   "testpass",
						IP:     net.ParseIP("127.1.1.1"),
						Action: conf.AuthActionRead,
						Path:   "mypath",
					})
					require.Error(t, err)

				case "wrong path":
					err := m.Authenticate(&Request{
						User:   "testuser",
						Pass:   "testpass",
						IP:     net.ParseIP("127.1.1.1"),
						Action: conf.AuthActionPublish,
						Path:   "wrong",
					})
					require.Error(t, err)
				}
			})
		}
	}
}

func TestAuthInternalRTSPDigest(t *testing.T) {
	m := Manager{
		Method: conf.AuthMethodInternal,
		InternalUsers: []conf.AuthInternalUser{
			{
				User: "myuser",
				Pass: "mypass",
				IPs:  conf.IPNetworks{mustParseCIDR("127.1.1.1/32")},
				Permissions: []conf.AuthInternalUserPermission{
					{
						Action: conf.AuthActionPublish,
						Path:   "mypath",
					},
				},
			},
		},
		HTTPAddress:     "",
		RTSPAuthMethods: []headers.AuthMethod{headers.AuthDigestMD5},
	}

	u, err := base.ParseURL("rtsp://127.0.0.1:8554/mypath")
	require.NoError(t, err)

	s, err := auth.NewSender(
		auth.GenerateWWWAuthenticate([]headers.AuthMethod{headers.AuthDigestMD5}, "IPCAM", "mynonce"),
		"myuser",
		"mypass",
	)
	require.NoError(t, err)

	req := &base.Request{
		Method: "ANNOUNCE",
		URL:    u,
	}

	s.AddAuthorization(req)

	err = m.Authenticate(&Request{
		IP:          net.ParseIP("127.1.1.1"),
		Action:      conf.AuthActionPublish,
		Path:        "mypath",
		RTSPRequest: req,
		RTSPNonce:   "mynonce",
	})
	require.NoError(t, err)
}

func TestAuthHTTP(t *testing.T) {
	for _, outcome := range []string{"ok", "fail"} {
		t.Run(outcome, func(t *testing.T) {
			m := Manager{
				Method:          conf.AuthMethodHTTP,
				HTTPAddress:     "http://127.0.0.1:9120/auth",
				RTSPAuthMethods: nil,
			}

			au := &testHTTPAuthenticator{}
			au.initialize(t, "rtsp", "publish")
			defer au.close()

			if outcome == "ok" {
				err := m.Authenticate(&Request{
					User:     "testpublisher",
					Pass:     "testpass",
					IP:       net.ParseIP("127.0.0.1"),
					Action:   conf.AuthActionPublish,
					Path:     "teststream",
					Protocol: ProtocolRTSP,
					Query:    "param=value",
				})
				require.NoError(t, err)
			} else {
				err := m.Authenticate(&Request{
					User:     "invalid",
					Pass:     "testpass",
					IP:       net.ParseIP("127.0.0.1"),
					Action:   conf.AuthActionPublish,
					Path:     "teststream",
					Protocol: ProtocolRTSP,
					Query:    "param=value",
				})
				require.Error(t, err)
			}
		})
	}
}

func TestAuthHTTPExclude(t *testing.T) {
	m := Manager{
		Method:      conf.AuthMethodHTTP,
		HTTPAddress: "http://not-to-be-used:9120/auth",
		HTTPExclude: []conf.AuthInternalUserPermission{{
			Action: conf.AuthActionPublish,
		}},
		RTSPAuthMethods: nil,
	}

	err := m.Authenticate(&Request{
		User:     "",
		Pass:     "",
		IP:       net.ParseIP("127.0.0.1"),
		Action:   conf.AuthActionPublish,
		Path:     "teststream",
		Protocol: ProtocolRTSP,
		Query:    "param=value",
	})
	require.NoError(t, err)
}
