package common

import (
	"context"
	"crypto/tls"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/defectus/glutton/pkg/iface"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// Run is the entry point to Glutton.
func Run() error {
	var (
		yamlConfiguration []byte
		err               error
	)
	// first see if we're configured by yaml
	file := flag.String("f", "", "configuration file path")
	debug := flag.Bool("d", false, "configuration file path")
	flag.Parse()

	if len(*file) > 0 {
		yamlConfiguration, err = os.ReadFile(*file)
		if err != nil {
			log.Panicf("error reading configuration file %s %+v", *file, err)
		}
	}
	env := CreateEnvironment(CreateConfiguration(new(iface.Configuration), *debug, yamlConfiguration), nil)
	if env.Configuration.Debug {
		log.Printf("current settings: %+v", env.Configuration)
	}
	defer closeAll(env)
	appContext, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	hookOnExit(cancelFunc)
	address := env.Configuration.Host + ":" + env.Configuration.Port
	tlsConfiguration, err := serverTLSConfig(env.Configuration)
	if err != nil {
		log.Panicf("error configuring tls %+v", err)
	}
	if tlsConfiguration != nil {
		log.Printf("listening on %s over https", address)
	} else {
		log.Printf("listening on %s", address)
	}
	return serve(appContext, env.Server, address, tlsConfiguration)
}

// closeAll releases all resources held by the environment.
func closeAll(env *iface.Env) {
	for _, closer := range env.Closers {
		if err := closer.Close(); err != nil {
			log.Printf("error closing %T: %+v", closer, err)
		}
	}
}

// server timeouts protecting the service from slow or stalled clients
const (
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 30 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 60 * time.Second
	shutdownTimeout   = 5 * time.Second
)

// serve runs the HTTP (or HTTPS when tlsConfig is provided) server and blocks until it fails or the provided context is cancelled. Any error returned by the server is reported to the caller instead of being discarded.
func serve(ctx context.Context, engine *gin.Engine, address string, tlsConfiguration *tls.Config) error {
	server := &http.Server{
		Addr:              address,
		Handler:           engine,
		TLSConfig:         tlsConfiguration,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
	serverError := make(chan error, 1)
	go func() {
		if tlsConfiguration != nil {
			// certificates come from TLSConfig, hence the empty file names
			serverError <- server.ListenAndServeTLS("", "")
			return
		}
		serverError <- server.ListenAndServe()
	}()
	select {
	case err := <-serverError:
		return errors.Wrap(err, "error running server")
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			return errors.Wrap(err, "error shutting down server")
		}
		return nil
	}
}

// hookOnExit listens for signal SIGHUP and once received it closes the provided `closing` channel.
func hookOnExit(cancelFunc context.CancelFunc) {
	go func(cancelFunc context.CancelFunc) {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Kill, os.Interrupt)
		<-signals
		log.Println("Initiating shutdown ...")
		cancelFunc()
	}(cancelFunc)
}

// initializeRoutes does the basic stuff needed to create a router.
func initializeRoutes(router *gin.Engine, env *iface.Env) *gin.RouterGroup {
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "HEAD", "PATCH"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "token"}
	router.Use(cors.New(config))
	v1 := router.Group("v1")
	glutton := v1.Group("glutton")
	return glutton
}

func renderError(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error(), "detail": err})
}
