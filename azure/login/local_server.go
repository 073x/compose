package login

import (
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

const failHTML = `
<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8" />
	<title>Login failed</title>
</head>
<body>
	<h4>Some failures occurred during the authentication</h4>
	<p>You can log an issue at <a href="https://github.com/azure/azure-cli/issues">Azure CLI GitHub Repository</a> and we will assist you in resolving it.</p>
</body>
</html>
`

const successHTML = `
<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8" />
	<meta http-equiv="refresh" content="10;url=https://docs.microsoft.com/cli/azure/">
	<title>Login successfully</title>
</head>
<body>
	<h4>You have logged into Microsoft Azure!</h4>
	<p>You can close this window, or we will redirect you to the <a href="https://docs.microsoft.com/cli/azure/">Azure CLI documents</a> in 10 seconds.</p>
</body>
</html>
`

const (
	// redirectHost is where the user's browser is redirected on authentication
	// completion. Note that only registered hosts can be used. i.e.:
	// "localhost" works but "127.0.0.1" does not.
	redirectHost = "localhost"
)

type localResponse struct {
	values url.Values
	err    error
}

// LocalServer is an Azure login server
type LocalServer struct {
	httpServer *http.Server
	listener   net.Listener
	queryCh    chan localResponse
}

// NewLocalServer creates an Azure login server
func NewLocalServer(queryCh chan localResponse) (*LocalServer, error) {
	s := &LocalServer{queryCh: queryCh}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handler())
	s.httpServer = &http.Server{Handler: mux}
	l, err := net.Listen("tcp", redirectHost+":0")
	if err != nil {
		return nil, err
	}
	s.listener = l
	p := l.Addr().(*net.TCPAddr).Port
	if p == 0 {
		return nil, errors.New("unable to allocate login server port")
	}
	return s, nil
}

// Serve starts the local Azure login server
func (s *LocalServer) Serve() {
	go func() {
		if err := s.httpServer.Serve(s.listener); err != nil {
			s.queryCh <- localResponse{
				err: errors.Wrap(err, "unable to start login server"),
			}
		}
	}()
}

// Close stops the local Azure login server
func (s *LocalServer) Close() {
	_ = s.httpServer.Close()
}

// Addr returns the address that the local Azure server is service to
func (s *LocalServer) Addr() string {
	return fmt.Sprintf("http://%s:%d", redirectHost, s.listener.Addr().(*net.TCPAddr).Port)
}

func (s *LocalServer) handler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, hasCode := r.URL.Query()["code"]; hasCode {
			if _, err := w.Write([]byte(successHTML)); err != nil {
				s.queryCh <- localResponse{
					err: errors.Wrap(err, "unable to write success HTML"),
				}
			} else {
				s.queryCh <- localResponse{values: r.URL.Query()}
			}
		} else {
			if _, err := w.Write([]byte(failHTML)); err != nil {
				s.queryCh <- localResponse{
					err: errors.Wrap(err, "unable to write fail HTML"),
				}
			} else {
				s.queryCh <- localResponse{values: r.URL.Query()}
			}
		}
	}
}
