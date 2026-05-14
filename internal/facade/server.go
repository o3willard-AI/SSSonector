package facade

import (
	"context"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"go.uber.org/zap"
)

const (
	// websocketGUID is the magic GUID used in the WebSocket protocol handshake
	// as defined in RFC 6455 Section 4.2.2
	websocketGUID = "258EAFA5-E914-47DA-95CA-5631BC565D11"

	// tunnelTokenHeader is the HTTP header carrying the HMAC-signed tunnel token
	tunnelTokenHeader = "X-Tunnel-Token"

	// defaultWebRoot is the default content returned for GET /.
	// This is the Apache2 Ubuntu default page — a convincing cover for the
	// HTTPS facade. The Ubuntu logo image has been removed since the facade
	// doesn't serve static assets; the rest is verbatim from the apache2 package.
	defaultWebRoot = `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
  <!--
    Modified from the Debian original for Ubuntu
    Last updated: 2022-03-22
    See: https://launchpad.net/bugs/1966004
  -->
  <head>
    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
    <title>Apache2 Ubuntu Default Page: It works</title>
    <style type="text/css" media="screen">
  * {
    margin: 0px 0px 0px 0px;
    padding: 0px 0px 0px 0px;
  }

  body, html {
    padding: 3px 3px 3px 3px;
    background-color: #D8DBE2;
    font-family: Ubuntu, Verdana, sans-serif;
    font-size: 11pt;
    text-align: center;
  }

  div.main_page {
    position: relative;
    display: table;
    width: 800px;
    margin-bottom: 3px;
    margin-left: auto;
    margin-right: auto;
    padding: 0px 0px 0px 0px;
    border-width: 2px;
    border-color: #212738;
    border-style: solid;
    background-color: #FFFFFF;
    text-align: center;
  }

  div.page_header {
    height: 100px;
    width: 100%;
    background-color: #F5F6F7;
  }

  div.page_header span {
    margin: 15px 0px 0px 50px;
    font-size: 180%;
    font-weight: bold;
  }

  div.banner {
    padding: 9px 6px 9px 6px;
    background-color: #E9510E;
    color: #FFFFFF;
    font-weight: bold;
    font-size: 112%;
    text-align: center;
    position: absolute;
    left: 40%;
    bottom: 30px;
    width: 20%;
  }

  div.table_of_contents {
    clear: left;
    min-width: 200px;
    margin: 3px 3px 3px 3px;
    background-color: #FFFFFF;
    text-align: left;
  }

  div.table_of_contents_item {
    clear: left;
    width: 100%;
    margin: 4px 0px 0px 0px;
    background-color: #FFFFFF;
    color: #000000;
    text-align: left;
  }

  div.table_of_contents_item a {
    margin: 6px 0px 0px 6px;
  }

  div.content_section {
    margin: 3px 3px 3px 3px;
    background-color: #FFFFFF;
    text-align: left;
  }

  div.content_section_text {
    padding: 4px 8px 4px 8px;
    color: #000000;
    font-size: 100%;
  }

  div.content_section_text pre {
    margin: 8px 0px 8px 0px;
    padding: 8px 8px 8px 8px;
    border-width: 1px;
    border-style: dotted;
    border-color: #000000;
    background-color: #F5F6F7;
    font-style: italic;
  }

  div.content_section_text p {
    margin-bottom: 6px;
  }

  div.content_section_text ul, div.content_section_text li {
    padding: 4px 8px 4px 16px;
  }

  div.section_header {
    padding: 3px 6px 3px 6px;
    background-color: #8E9CB2;
    color: #FFFFFF;
    font-weight: bold;
    font-size: 112%;
    text-align: center;
  }

  div.section_header_grey {
    background-color: #9F9386;
  }

  .floating_element {
    position: relative;
    float: left;
  }

  div.table_of_contents_item a,
  div.content_section_text a {
    text-decoration: none;
    font-weight: bold;
  }

  div.table_of_contents_item a:link,
  div.table_of_contents_item a:visited,
  div.table_of_contents_item a:active {
    color: #000000;
  }

  div.table_of_contents_item a:hover {
    background-color: #000000;
    color: #FFFFFF;
  }

  div.content_section_text a:link,
  div.content_section_text a:visited,
   div.content_section_text a:active {
    background-color: #DCDFE6;
    color: #000000;
  }

  div.content_section_text a:hover {
    background-color: #000000;
    color: #DCDFE6;
  }

  div.validator {
  }
    </style>
  </head>
  <body>
    <div class="main_page">
      <div class="page_header floating_element">
        <div style="padding-top: 1.5em;">
          <span class="floating_element">
            Apache2 Default Page
          </span>
        </div>
        <div class="banner">
          <div id="about"></div>
          It works!
        </div>
      </div>
      <div class="content_section floating_element">
        <div class="content_section_text">
          <p>
                This is the default welcome page used to test the correct 
                operation of the Apache2 server after installation on Ubuntu systems.
                It is based on the equivalent page on Debian, from which the Ubuntu Apache
                packaging is derived.
                If you can read this page, it means that the Apache HTTP server installed at
                this site is working properly. You should <b>replace this file</b> (located at
                <tt>/var/www/html/index.html</tt>) before continuing to operate your HTTP server.
          </p>

          <p>
                If you are a normal user of this web site and don't know what this page is
                about, this probably means that the site is currently unavailable due to
                maintenance.
                If the problem persists, please contact the site's administrator.
          </p>

        </div>
        <div class="section_header">
          <div id="changes"></div>
                Configuration Overview
        </div>
        <div class="content_section_text">
          <p>
                Ubuntu's Apache2 default configuration is different from the
                upstream default configuration, and split into several files optimized for
                interaction with Ubuntu tools. The configuration system is
                <b>fully documented in
                /usr/share/doc/apache2/README.Debian.gz</b>. Refer to this for the full
                documentation. Documentation for the web server itself can be
                found by accessing the <a href="/manual">manual</a> if the <tt>apache2-doc</tt>
                package was installed on this server.
          </p>
          <p>
                The configuration layout for an Apache2 web server installation on Ubuntu systems is as follows:
          </p>
          <pre>
/etc/apache2/
|-- apache2.conf
|       `--  ports.conf
|-- mods-enabled
|       |-- *.load
|       `-- *.conf
|-- conf-enabled
|       `-- *.conf
|-- sites-enabled
|       `-- *.conf
          </pre>
          <ul>
                        <li>
                           <tt>apache2.conf</tt> is the main configuration
                           file. It puts the pieces together by including all remaining configuration
                           files when starting up the web server.
                        </li>

                        <li>
                           <tt>ports.conf</tt> is always included from the
                           main configuration file. It is used to determine the listening ports for
                           incoming connections, and this file can be customized anytime.
                        </li>

                        <li>
                           Configuration files in the <tt>mods-enabled/</tt>,
                           <tt>conf-enabled/</tt> and <tt>sites-enabled/</tt> directories contain
                           particular configuration snippets which manage modules, global configuration
                           fragments, or virtual host configurations, respectively.
                        </li>

                        <li>
                           They are activated by symlinking available
                           configuration files from their respective
                           *-available/ counterparts. These should be managed
                           by using our helpers
                           <tt>
                                a2enmod,
                                a2dismod,
                           </tt>
                           <tt>
                                a2ensite,
                                a2dissite,
                            </tt>
                                and
                           <tt>
                                a2enconf,
                                a2disconf
                           </tt>. See their respective man pages for detailed information.
                        </li>

                        <li>
                           The binary is called apache2 and is managed using systemd, so to
                           start/stop the service use <tt>systemctl start apache2</tt> and
                           <tt>systemctl stop apache2</tt>, and use <tt>systemctl status apache2</tt>
                           and <tt>journalctl -u apache2</tt> to check status.  <tt>system</tt>
                           and <tt>apache2ctl</tt> can also be used for service management if
                           desired.
                           <b>Calling <tt>/usr/bin/apache2</tt> directly will not work</b> with the
                           default configuration.
                        </li>
          </ul>
        </div>

        <div class="section_header">
            <div id="docroot"></div>
                Document Roots
        </div>

        <div class="content_section_text">
            <p>
                By default, Ubuntu does not allow access through the web browser to
                <em>any</em> file outside of those located in <tt>/var/www</tt>,
                <a href="http://httpd.apache.org/docs/2.4/mod/mod_userdir.html" rel="nofollow">public_html</a>
                directories (when enabled) and <tt>/usr/share</tt> (for web
                applications). If your site is using a web document root
                located elsewhere (such as in <tt>/srv</tt>) you may need to whitelist your
                document root directory in <tt>/etc/apache2/apache2.conf</tt>.
            </p>
            <p>
                The default Ubuntu document root is <tt>/var/www/html</tt>. You
                can make your own virtual hosts under /var/www.
            </p>
        </div>

        <div class="section_header">
          <div id="bugs"></div>
                Reporting Problems
        </div>
        <div class="content_section_text">
          <p>
                Please use the <tt>ubuntu-bug</tt> tool to report bugs in the
                Apache2 package with Ubuntu. However, check <a
                href="https://bugs.launchpad.net/ubuntu/+source/apache2"
                rel="nofollow">existing bug reports</a> before reporting a new bug.
          </p>
          <p>
                Please report bugs specific to modules (such as PHP and others)
                to their respective packages, not to the web server itself.
          </p>
        </div>

      </div>
    </div>
    <div class="validator">
    </div>
  </body>
</html>`

	// connectPath is the path used for tunnel negotiation
	connectPath = "/connect"
)

// Server represents the HTTPS facade server.
// It serves a legitimate-looking website on port 443 while also handling
// tunnel connection upgrades disguised as WebSocket connections.
type Server struct {
	config     *types.FacadeConfig
	authConfig *types.AuthConfig
	logger     *zap.Logger
	httpServer *http.Server
	listener   net.Listener
	secret     []byte
	tokenTTL   time.Duration
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup

	// tunnelPorts is the set of valid tunnel ports this facade routes to
	tunnelPorts map[int]bool
}

// NewServer creates a new HTTPS facade server.
func NewServer(cfg *types.FacadeConfig, authCfg *types.AuthConfig, logger *zap.Logger) (*Server, error) {
	if cfg == nil {
		return nil, fmt.Errorf("facade config is required")
	}
	if authCfg == nil {
		return nil, fmt.Errorf("auth config is required")
	}

	// Resolve the token secret
	caFile := cfg.TLS.CAFile
	if caFile == "" {
		caFile = authCfg.CAFile
	}
	secret, err := ResolveSecret(cfg.TokenSecret, caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve token secret: %w", err)
	}

	// Build tunnel ports set
	tunnelPorts := make(map[int]bool, len(cfg.TunnelPorts))
	for _, port := range cfg.TunnelPorts {
		tunnelPorts[port] = true
	}

	tokenTTL := cfg.TokenTTL
	if tokenTTL <= 0 {
		tokenTTL = DefaultTokenTTL
	}

	ctx, cancel := context.WithCancel(context.Background())

	s := &Server{
		config:      cfg,
		authConfig:  authCfg,
		logger:      logger,
		secret:      secret,
		tokenTTL:    tokenTTL,
		tunnelPorts: tunnelPorts,
		ctx:         ctx,
		cancel:      cancel,
	}

	return s, nil
}

// Start starts the HTTPS facade server.
func (s *Server) Start() error {
	// Build TLS config
	tlsConfig, err := s.buildTLSConfig()
	if err != nil {
		return fmt.Errorf("failed to build TLS config: %w", err)
	}

	// Create HTTP mux
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRoot)
	mux.HandleFunc(connectPath, s.handleConnect)

	listenAddr := fmt.Sprintf("%s:%d", s.config.ListenAddress, s.config.ListenPort)

	s.httpServer = &http.Server{
		Addr:      listenAddr,
		Handler:   mux,
		TLSConfig: tlsConfig,
		// Timeouts to prevent slowloris attacks
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Create the TLS listener
	ln, err := tls.Listen("tcp", listenAddr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to start facade listener on %s: %w", listenAddr, err)
	}
	s.listener = ln

	s.logger.Info("HTTPS facade started",
		zap.String("address", listenAddr),
		zap.Int("tunnel_ports", len(s.tunnelPorts)),
	)

	// Serve in a goroutine
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Facade server error", zap.Error(err))
		}
	}()

	return nil
}

// Stop gracefully stops the HTTPS facade server.
func (s *Server) Stop() error {
	s.logger.Info("Stopping HTTPS facade")
	s.cancel()

	// Give active connections time to finish
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		s.logger.Error("Facade shutdown error", zap.Error(err))
		// Force close
		s.httpServer.Close()
	}

	s.wg.Wait()
	return nil
}

// handleRoot serves a legitimate-looking web page for GET /.
// This makes the server appear as a normal website to casual inspection,
// port scanners, and automated probes.
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	// Only serve the root path exactly -- everything else is 404
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	content := s.config.WebRoot
	if content == "" {
		content = defaultWebRoot
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Server", "nginx") // Blend in with common web servers
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		fmt.Fprint(w, content)
	}
}

// handleConnect handles tunnel connection upgrade requests.
// It validates the WebSocket upgrade headers and HMAC token, then hijacks
// the connection and proxies it to the appropriate local tunnel port.
func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	// Verify this is a WebSocket upgrade request
	if !isWebSocketUpgrade(r) {
		// Not a WebSocket upgrade -- return 404 to look like a normal server
		http.NotFound(w, r)
		return
	}

	// Extract and validate the tunnel token
	tokenStr := r.Header.Get(tunnelTokenHeader)
	if tokenStr == "" {
		s.logger.Debug("Upgrade request missing tunnel token",
			zap.String("remote", r.RemoteAddr),
		)
		http.NotFound(w, r)
		return
	}

	port, err := ValidateToken(tokenStr, s.secret, s.tokenTTL)
	if err != nil {
		s.logger.Debug("Invalid tunnel token",
			zap.String("remote", r.RemoteAddr),
			zap.Error(err),
		)
		// Return 404 instead of 403 to avoid leaking information
		http.NotFound(w, r)
		return
	}

	// Verify the port is in our allowed list
	if !s.tunnelPorts[port] {
		s.logger.Warn("Token for unconfigured tunnel port",
			zap.String("remote", r.RemoteAddr),
			zap.Int("port", port),
		)
		http.NotFound(w, r)
		return
	}

	// Compute the WebSocket accept value
	wsKey := r.Header.Get("Sec-WebSocket-Key")
	if wsKey == "" {
		http.NotFound(w, r)
		return
	}
	wsAccept := computeWebSocketAccept(wsKey)

	// Hijack the connection
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		s.logger.Error("HTTP server does not support hijacking")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	conn, buf, err := hijacker.Hijack()
	if err != nil {
		s.logger.Error("Failed to hijack connection",
			zap.String("remote", r.RemoteAddr),
			zap.Error(err),
		)
		return
	}

	// Send the 101 Switching Protocols response manually
	response := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + wsAccept + "\r\n" +
		"\r\n"

	if _, err := buf.WriteString(response); err != nil {
		s.logger.Error("Failed to write upgrade response",
			zap.String("remote", r.RemoteAddr),
			zap.Error(err),
		)
		conn.Close()
		return
	}
	if err := buf.Flush(); err != nil {
		s.logger.Error("Failed to flush upgrade response",
			zap.String("remote", r.RemoteAddr),
			zap.Error(err),
		)
		conn.Close()
		return
	}

	s.logger.Info("Tunnel upgrade accepted",
		zap.String("remote", r.RemoteAddr),
		zap.Int("tunnel_port", port),
	)

	// Proxy the hijacked connection to the local tunnel port
	tunnelAddr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := Proxy(s.ctx, conn, tunnelAddr, s.logger); err != nil {
			s.logger.Debug("Proxy ended",
				zap.String("remote", r.RemoteAddr),
				zap.Int("tunnel_port", port),
				zap.Error(err),
			)
		}
	}()
}

// buildTLSConfig creates the TLS configuration for the facade server.
func (s *Server) buildTLSConfig() (*tls.Config, error) {
	// Resolve certificate paths -- facade TLS config takes priority, then auth config
	certFile := s.config.TLS.CertFile
	if certFile == "" {
		certFile = s.authConfig.CertFile
	}
	keyFile := s.config.TLS.KeyFile
	if keyFile == "" {
		keyFile = s.authConfig.KeyFile
	}
	caFile := s.config.TLS.CAFile
	if caFile == "" {
		caFile = s.authConfig.CAFile
	}

	if certFile == "" || keyFile == "" {
		return nil, fmt.Errorf("certificate and key files are required for the facade")
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load facade certificate: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		// Use standard cipher suites that match typical web servers
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
		PreferServerCipherSuites: true,
	}

	// Load CA for optional client certificate verification
	// Note: The facade does NOT require mTLS -- authentication is via HMAC tokens.
	// However, if a CA is configured, we can use it to verify client certs as an
	// additional layer of security.
	if caFile != "" {
		caCert, err := os.ReadFile(caFile)
		if err != nil {
			s.logger.Warn("Failed to read CA file for facade, continuing without client cert verification",
				zap.Error(err),
			)
		} else {
			caPool := x509.NewCertPool()
			if caPool.AppendCertsFromPEM(caCert) {
				tlsConfig.ClientCAs = caPool
				// VerifyClientCertIfGiven allows both browser clients (no cert)
				// and tunnel clients (with cert) to connect
				tlsConfig.ClientAuth = tls.VerifyClientCertIfGiven
			}
		}
	}

	return tlsConfig, nil
}

// isWebSocketUpgrade checks if the request is a valid WebSocket upgrade request.
func isWebSocketUpgrade(r *http.Request) bool {
	if r.Method != http.MethodGet {
		return false
	}

	// Check Connection: Upgrade header (case-insensitive)
	connection := r.Header.Get("Connection")
	hasUpgrade := false
	for _, v := range strings.Split(connection, ",") {
		if strings.TrimSpace(strings.ToLower(v)) == "upgrade" {
			hasUpgrade = true
			break
		}
	}
	if !hasUpgrade {
		return false
	}

	// Check Upgrade: websocket header (case-insensitive)
	upgrade := r.Header.Get("Upgrade")
	if strings.ToLower(strings.TrimSpace(upgrade)) != "websocket" {
		return false
	}

	// Check Sec-WebSocket-Version: 13
	if r.Header.Get("Sec-WebSocket-Version") != "13" {
		return false
	}

	// Check Sec-WebSocket-Key is present
	if r.Header.Get("Sec-WebSocket-Key") == "" {
		return false
	}

	return true
}

// computeWebSocketAccept computes the Sec-WebSocket-Accept value per RFC 6455.
func computeWebSocketAccept(key string) string {
	h := sha1.New()
	h.Write([]byte(key + websocketGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
