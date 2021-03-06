// Package logger handles logging for JURA.
//
// This is a thing wrapper around inconshreveable's log15. It
// additionally brings a configuration format (from YAML) with the
// ability to log to several destinations.
//
// It also brings some conventions like the presence of "module" in
// each context to be able to filter logs more easily. However, this
// convention is not really enforced. Once you have a root logger,
// create sublogger with New and provide a new value for "module".
package logger

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "gopkg.in/inconshreveable/log15.v2"
	"gopkg.in/inconshreveable/log15.v2/stack"
)

const (
	timeFormat = "2006-01-02T15:04:05-0700"
)

// New creates a new logger from a configuration.
func New(config Configuration, additionalHandler log.Handler, prefix string) (log.Logger, error) {
	handlers := make([]log.Handler, 0, 10)
	// We need to build the appropriate handler.
	if config.Console {
		handlers = append(handlers, log.StdoutHandler)
	}
	if config.Syslog {
		handler, err := log.SyslogHandler("lrg", log.LogfmtFormat())
		if err != nil {
			return nil, errors.Wrap(err, "unable to open syslog connection")
		}
		handlers = append(handlers, handler)
	}
	for _, logFile := range config.Files {
		var formatter log.Format
		switch logFile.Format {
		case FormatPlain:
			formatter = log.LogfmtFormat()
		case FormatJSON:
			formatter = JSONv1Format()
		default:
			panic(fmt.Sprintf("unknown format provided: %v", logFile.Format))
		}
		handler, err := log.FileHandler(logFile.Name, formatter)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to open log file %q", logFile.Name)
		}
		handlers = append(handlers, handler)
	}

	// Initialize the logger
	var logger = log.New()
	logHandler := log.LvlFilterHandler(
		log.Lvl(config.Level),
		contextHandler(log.MultiHandler(handlers...), prefix))
	if additionalHandler != nil {
		logHandler = log.MultiHandler(logHandler, additionalHandler)
	}
	logger.SetHandler(logHandler)
	return logger, nil
}

// Add more context to log entry. This is similar to
// log.CallerFileHandler and log.CallerFuncHandler but it's a bit
// smarter on how the stack trace is inspected to avoid logging
// modules. It adds a "caller" (qualified function + line number) and
// a "module" (JURA module name).
func contextHandler(h log.Handler, prefix string) log.Handler {
	return log.FuncHandler(func(r *log.Record) error {
		callStack := stack.Callers().TrimBelow(stack.Call(r.CallPC[0]))
		callerFound := false
		for _, call := range callStack {
			if !callerFound {
				// Searching for the first caller.
				caller := fmt.Sprintf("%+v", call)
				if !strings.HasPrefix(caller, fmt.Sprintf("%s/reporter", prefix)) {
					r.Ctx = append(r.Ctx, "caller", caller)
					callerFound = true
				}
			}
			if callerFound {
				// Searching for the module name
				module := fmt.Sprintf("%+n", call)
				if !strings.HasPrefix(module, prefix) {
					continue
				}
				if strings.HasPrefix(module, fmt.Sprintf("%s/vendor/", prefix)) {
					continue
				}
				module = strings.SplitN(module, ".", 2)[0]
				r.Ctx = append(r.Ctx, "module", module)
				return h.Log(r)
			}
		}
		return h.Log(r)
	})
}

// JSONv1Format formats log records as JSONv1 objects separated by newlines.
func JSONv1Format() log.Format {
	return log.FormatFunc(func(r *log.Record) []byte {
		props := make(map[string]interface{})

		var level string
		switch r.Lvl {
		case log.LvlDebug:
			level = "debug"
		case log.LvlInfo:
			level = "info"
		case log.LvlWarn:
			level = "warn"
		case log.LvlError:
			level = "error"
		case log.LvlCrit:
			level = "crit"
		default:
			panic("bad level")
		}

		props["@version"] = 1
		props["@timestamp"] = r.Time
		props["level"] = level
		props["message"] = r.Msg

		for i := 0; i < len(r.Ctx); i += 2 {
			k, ok := r.Ctx[i].(string)
			if !ok {
				props["JURALOG_ERROR"] = fmt.Sprintf("%+v is not a string key", r.Ctx[i])
			}
			props[k] = formatJSONValue(r.Ctx[i+1])
		}

		b, err := json.Marshal(props)
		if err != nil {
			b, _ = json.Marshal(map[string]string{
				"JURALOG_ERROR": err.Error(),
			})
		}

		b = append(b, '\n')
		return b
	})

}

func formatJSONValue(value interface{}) interface{} {
	switch v := value.(type) {
	case time.Time:
		return v.Format(timeFormat)
	case error:
		return v.Error()
	case fmt.Stringer:
		return v.String()
	case int, int8, int16, int32, int64, float32, float64, uint, uint8, uint16, uint32, uint64, string:
		return value
	default:
		return fmt.Sprintf("%+v", value)
	}
}
