package capabilities

import (
	"github.com/pkg/errors"
	"github.com/suborbital/vektor/vlog"
)

var ErrCapabilityNotAvailable = errors.New("capability not available")

// Capabilities define the capabilities available to a Runnable
type Capabilities struct {
	config CapabilityConfig

	Auth          AuthCapability
	LoggerSource  LoggerCapability
	HTTPClient    HTTPCapability
	GraphQLClient GraphQLCapability
	Cache         CacheCapability
	FileSource    FileCapability
	Database      DatabaseCapability

	// RequestHandler and doFunc are special because they are more
	// sensitive; they could cause memory leaks or expose internal state,
	// so they cannot be swapped out for a different implementation.
	RequestConfig *RequestHandlerConfig
}

// New returns the default capabilities with the provided Logger
func New(logger *vlog.Logger) *Capabilities {
	// this will never error with the default config, as the db capability is disabled
	caps, _ := NewWithConfig(DefaultConfigWithLogger(logger))

	return caps
}

func NewWithConfig(config CapabilityConfig) (*Capabilities, error) {
	database, err := NewSqlDatabase(config.DB)
	if err != nil {
		return nil, errors.Wrap(err, "failed to NewSqlDatabase")
	}

	caps := &Capabilities{
		config:        config,
		Auth:          DefaultAuthProvider(*config.Auth),
		LoggerSource:  DefaultLoggerSource(*config.Logger),
		HTTPClient:    DefaultHTTPClient(*config.HTTP),
		GraphQLClient: DefaultGraphQLClient(*config.GraphQL),
		Cache:         SetupCache(*config.Cache),
		FileSource:    DefaultFileSource(*config.File),
		Database:      database,

		// RequestHandler and doFunc don't get set here since they are set by
		// the rt and rwasm internals; a better solution for this should probably be found
		RequestConfig: config.Request,
	}

	return caps, nil
}

// Config returns the configuration that was used to create the Capabilities
// the config cannot be changed, but it can be used to determine what was
// previously set so that the orginal config (like enabled settings) can be respected
func (c Capabilities) Config() CapabilityConfig {
	return c.config
}
