package common

import (
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"strconv"

	yaml "gopkg.in/yaml.v2"

	"github.com/defectus/glutton/pkg/auth"
	"github.com/defectus/glutton/pkg/handler"
	"github.com/defectus/glutton/pkg/iface"
	"github.com/defectus/glutton/pkg/notifier"
	"github.com/defectus/glutton/pkg/parser"
	"github.com/defectus/glutton/pkg/saver"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// CreateConfiguration makes a configuration by creating or enhancing an existing one. Values are taken from the yaml file first (when supplied) and from environment variables last, so environment variables have the highest precedence. Fields that are neither configured nor set default to the value of their `default` struct tag.
func CreateConfiguration(configuration *iface.Configuration, debug bool, yamlConfiguration []byte) *iface.Configuration {
	if configuration == nil {
		configuration = new(iface.Configuration)
	}
	if err := applyDefaultTags(configuration); err != nil {
		log.Panicf("createConfigration: failed to apply defaults %+v", err)
	}
	if len(yamlConfiguration) > 0 {
		if err := yaml.Unmarshal(yamlConfiguration, configuration); err != nil {
			log.Printf("createConfigration: error parsing configuration %+v", err)
		}
	}
	if err := applyEnv(configuration); err != nil {
		log.Panicf("createConfigration: failed to read configuration from environment %+v", err)
	}
	configuration.Debug = configuration.Debug || debug
	// if the yaml file did not define any route fall back to the single environment based route
	if len(configuration.Settings) == 0 {
		settings := new(iface.Settings)
		if err := valueFromEnvVar(settings); err != nil {
			log.Panicf("createConfigration: failed to read settings %+v", err)
		}
		configuration.Settings = append(configuration.Settings, *settings)
	}
	return configuration
}

// CreateEnvironment prepares the whole environment - provided with a configuration it creates all structs and handlers.
func CreateEnvironment(configuration *iface.Configuration, env *iface.Env) *iface.Env {
	if env == nil {
		env = &iface.Env{
			Savers:    map[string]reflect.Type{},
			Notifiers: map[string]reflect.Type{},
			Parsers:   map[string]reflect.Type{},
		}
	}
	env.Configuration = configuration
	if !configuration.Debug {
		gin.SetMode(gin.ReleaseMode)
	}
	env.Server = gin.Default()
	registerCompoments(env)
	gluttonRoute := initializeRoutes(env.Server, env)
	for index := range env.Configuration.Settings {
		settings := &env.Configuration.Settings[index]
		applyDefaults(settings)
		var (
			instance interface{}
			notifier iface.PayloadNotifier
			saver    iface.PayloadSaver
			parser   iface.PayloadParser
			err      error
			ok       bool
		)
		if len(settings.Notifier) > 0 {
			instance, err = createInstanceOf(env.Notifiers, settings.Notifier, settings)
			if err != nil {
				log.Panicf("error creating notifier %+v", err)
			}
			if notifier, ok = instance.(iface.PayloadNotifier); !ok {
				log.Panicf("exptected notifier, got %s", reflect.TypeOf(instance))
			}
		}
		if len(settings.Saver) > 0 {
			instance, err = createInstanceOf(env.Savers, settings.Saver, settings)
			if err != nil {
				log.Panicf("error creating saver %+v", err)
			}
			if saver, ok = instance.(iface.PayloadSaver); !ok {
				log.Panicf("exptected saver, got %s", reflect.TypeOf(instance))
			}
			if closer, ok := instance.(io.Closer); ok {
				env.Closers = append(env.Closers, closer)
			}
		}
		if len(settings.Parser) > 0 {
			instance, err = createInstanceOf(env.Parsers, settings.Parser, settings)
			if err != nil {
				log.Panicf("error creating parser %+v", err)
			}
			if parser, ok = instance.(iface.PayloadParser); !ok {
				log.Panicf("exptected parser, got %s", reflect.TypeOf(instance))
			}
		}
		h := handler.CreateHandler(settings.URI, parser, notifier, saver, settings.Debug)
		if settings.UseToken {
			key := []byte(settings.TokenKey)
			if err := auth.ValidateKey(key); err != nil {
				log.Panicf("invalid token configuration for uri %s: %+v", settings.URI, err)
			}
			h = handler.ValidateTokenHandler(h, settings.URI, key, configuration.Debug)
			gluttonRoute.GET(settings.URI+"/token", handler.CreateTokenHandler(settings.URI, key, configuration.Debug))
		}
		gluttonRoute.POST(settings.URI, handler.RedirectHandler(h, http.StatusFound, settings.Redirect))
	}
	return env
}

// applyDefaults fills empty fields of the provided settings with the values declared in their `default` struct tags. Settings coming from a yaml file bypass the environment based defaults, so without this they would end up with empty component names and paths.
func applyDefaults(settings *iface.Settings) {
	value := reflect.ValueOf(settings).Elem()
	typ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		def := typ.Field(i).Tag.Get("default")
		switch field.Kind() {
		case reflect.String:
			if len(field.String()) == 0 {
				field.SetString(def)
			}
		case reflect.Int:
			if field.Int() == 0 {
				in, _ := strconv.ParseInt(def, 10, 64)
				field.SetInt(in)
			}
		}
	}
}

func registerCompoments(env *iface.Env) {
	env.Notifiers["NilNotifier"] = reflect.TypeOf(notifier.NilNotifier{})
	env.Notifiers["SMTPNotifier"] = reflect.TypeOf(notifier.SMTPNotifier{})
	env.Savers["SimpleFileSystemSaver"] = reflect.TypeOf(saver.SimpleFileSystemSaver{})
	env.Savers["DatabaseSaver"] = reflect.TypeOf(saver.DatabaseSaver{})
	env.Parsers["SimpleParser"] = reflect.TypeOf(parser.SimpleParser{})
}

// createInstanceOf creates an instance of given name and configures it with the given settings (if implements the Configurable interface).
func createInstanceOf(types map[string]reflect.Type, name string, settings *iface.Settings) (interface{}, error) {
	if _, found := types[name]; !found {
		return nil, errors.Errorf("type not found error configuring instance %s with types %+v", name, types)
	}
	v := reflect.New(types[name])
	if c, ok := v.Interface().(iface.Configurable); ok {
		err := c.Configure(settings)
		if err != nil {
			return nil, errors.Wrapf(err, "error configuring instance %s with %+v", name, settings)
		}
	}
	return v.Interface(), nil
}

// valueFromEnvVar recursively traverses the supplied variable (pointer to a structure) and assigns values based on each field's `env` tag. Should the corresponding environment variable be empty the `default` tag's value is used.
// Note that strings, bools and ints are supported at the moment.
func valueFromEnvVar(value interface{}) error {
	if err := applyDefaultTags(value); err != nil {
		return err
	}
	return applyEnv(value)
}

// applyDefaultTags fills zero valued fields of the supplied structure from their `default` struct tag.
func applyDefaultTags(value interface{}) error {
	val, err := dereference(value)
	if err != nil {
		return err
	}
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		def := typ.Field(i).Tag.Get("default")
		switch field.Kind() {
		case reflect.String:
			if len(field.String()) == 0 {
				field.SetString(def)
			}
		case reflect.Int:
			if field.Int() == 0 {
				in, _ := strconv.ParseInt(def, 10, 64)
				field.SetInt(in)
			}
		case reflect.Bool:
			if bo, _ := strconv.ParseBool(def); !field.Bool() && bo {
				field.SetBool(true)
			}
		case reflect.Ptr:
			if field.Type().Elem().Kind() == reflect.Struct {
				if err := applyDefaultTags(field.Interface()); err != nil {
					return errors.Wrapf(err, "error processing %s", typ.Field(i).Name)
				}
			}
		}
	}
	return nil
}

// applyEnv overrides fields of the supplied structure with the values of the environment variables referenced by their `env` tag. Fields without a set environment variable are left untouched.
func applyEnv(value interface{}) error {
	val, err := dereference(value)
	if err != nil {
		return err
	}
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if field.Kind() == reflect.Ptr {
			if field.Type().Elem().Kind() == reflect.Struct {
				if err := applyEnv(field.Interface()); err != nil {
					return errors.Wrapf(err, "error processing %s", typ.Field(i).Name)
				}
			}
			continue
		}
		tag := typ.Field(i).Tag.Get("env")
		if len(tag) == 0 {
			tag = typ.Field(i).Name
		}
		env, present := os.LookupEnv(tag)
		if !present || len(env) == 0 {
			continue
		}
		switch field.Kind() {
		case reflect.String:
			field.SetString(env)
		case reflect.Int:
			in, _ := strconv.ParseInt(env, 10, 64)
			field.SetInt(in)
		case reflect.Bool:
			bo, _ := strconv.ParseBool(env)
			field.SetBool(bo)
		default:
			log.Printf("applyEnv: unsupported kind %s at %s.", field.Kind(), typ.Field(i).Name)
		}
	}
	return nil
}

// dereference returns the structure pointed to by value, or an error if it is not a pointer to a structure.
func dereference(value interface{}) (reflect.Value, error) {
	val := reflect.ValueOf(value)
	if val.Kind() != reflect.Ptr {
		return val, errors.New("only pointer type values are supported")
	}
	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return val, errors.New("only struct types are supported")
	}
	return val, nil
}
