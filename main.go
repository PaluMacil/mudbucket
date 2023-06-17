package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"embed"
	"encoding/pem"
	"flag"
	"fmt"
	"golang.org/x/exp/slog"
	"html/template"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

//go:embed *.gohtml
var templates embed.FS

// ServerConfig holds configuration data for the server
type ServerConfig struct {
	Port      string
	StaticDir string
	CertDir   string
	Token     string
	Secure    bool
}

func main() {
	// Load server configuration from environment
	cfg := loadConfigurationFromEnv()

	// Register HTTP handlers
	cfg.registerHandlers()

	// Start the server
	cfg.startServer()
}

// loadConfigurationFromEnv loads the server configuration from the environment
func loadConfigurationFromEnv() *ServerConfig {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8483"
	}

	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = filepath.Join(".", "static")
	}

	certDir := os.Getenv("CERT_DIR")
	if certDir == "" {
		certDir = filepath.Join(".", "certs")
	}

	token := os.Getenv("TOKEN_VALUE")
	if token == "" {
		token = "token123"
		slog.Info("Warning: Using default token")
	}

	secure := flag.Bool("secure", false, "Enable secure server")
	flag.Parse()

	return &ServerConfig{
		Port:      port,
		StaticDir: staticDir,
		CertDir:   certDir,
		Token:     token,
		Secure:    *secure,
	}
}

// registerHandlers registers HTTP handlers for the server
func (cfg *ServerConfig) registerHandlers() {
	http.HandleFunc("/", cfg.rootHandler)
	http.HandleFunc("/login", cfg.loginHandler)
	http.HandleFunc("/logout", cfg.logoutHandler)
}

// startServer starts the server with the provided configuration
func (cfg *ServerConfig) startServer() {
	slog.Info(fmt.Sprintf("Starting server at port %s", cfg.Port))

	server := &http.Server{
		Addr: ":" + cfg.Port,
	}

	if cfg.Secure {
		certPath, keyPath, err := generateSelfSignedCert(cfg.CertDir)
		if err != nil {
			slog.Error(fmt.Sprintf("Error generating certificate: %v", err))
			os.Exit(1)
		}

		server.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS13,
			CurvePreferences: []tls.CurveID{
				tls.CurveP256,
				tls.X25519,
			},
		}

		err = server.ListenAndServeTLS(certPath, keyPath)
		if err != nil {
			slog.Error(fmt.Sprintf("Error starting secure server: %v", err))
			os.Exit(1)
		}
	} else {
		err := server.ListenAndServe()
		if err != nil {
			slog.Error(fmt.Sprintf("Error starting server: %v", err))
			os.Exit(1)
		}
	}
	slog.Info("Server shutting down")
}

func (cfg *ServerConfig) rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tokenCookie, staticDirReadErr := r.Cookie("token")
	if staticDirReadErr != nil || tokenCookie.Value != cfg.Token {
		renderLoginForm(w, false)
		return
	}

	// If there's a path, try to serve the file from the static directory
	if r.URL.Path != "/" {
		fs := http.FileServer(http.Dir(cfg.StaticDir))
		http.StripPrefix("/", fs).ServeHTTP(w, r)
		return
	}

	// If there's no path, list the files

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	files, staticDirReadErr := os.ReadDir(cfg.StaticDir)
	if staticDirReadErr != nil {
		readErrMsg := fmt.Sprintf("Error reading directory: %v", staticDirReadErr)
		slog.Error(readErrMsg)
	}

	tmpl, err := template.ParseFS(templates, "root.gohtml")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Files       []os.DirEntry
		ShowWarning bool
	}{
		Files:       files,
		ShowWarning: staticDirReadErr != nil,
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (cfg *ServerConfig) loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if r.FormValue("token") != cfg.Token {
		renderLoginForm(w, true)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:   "token",
		Value:  r.FormValue("token"),
		Path:   "/",
		Domain: r.Host,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func renderLoginForm(w http.ResponseWriter, showWarning bool) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	tmpl, err := template.ParseFS(templates, "login.gohtml")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := struct {
		ShowWarning bool
	}{
		ShowWarning: showWarning,
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

}

func (cfg *ServerConfig) logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   "token",
		Value:  "",
		Path:   "/",
		Domain: r.Host,
		MaxAge: -1,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func generateSelfSignedCert(certDir string) (certPath string, keyPath string, err error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"My Company"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		return "", "", err
	}

	os.MkdirAll(certDir, os.ModePerm)
	certPath = filepath.Join(certDir, "cert.pem")
	keyPath = filepath.Join(certDir, "key.pem")

	certOut, err := os.Create(certPath)
	if err != nil {
		return "", "", err
	}
	defer certOut.Close()
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	b, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return "", "", err
	}

	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", "", err
	}
	defer keyOut.Close()
	pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: b})

	return certPath, keyPath, nil
}
