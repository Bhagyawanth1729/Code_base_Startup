package logger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Logger struct {
	*zerolog.Logger
}

func New(level string, pretty bool) *Logger {
	var l zerolog.Logger

	if pretty {
		output := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
		l = zerolog.New(output).With().Timestamp().Logger()
	} else {
		l = zerolog.New(os.Stdout).With().Timestamp().Logger()
	}

	parsedLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		parsedLevel = zerolog.InfoLevel
	}
	l = l.Level(parsedLevel)

	logger := &Logger{Logger: &l}
	log.Logger = l

	return logger
}

func NewWithWriter(w io.Writer, level string) *Logger {
	l := zerolog.New(w).With().Timestamp().Logger()

	parsedLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		parsedLevel = zerolog.InfoLevel
	}
	l = l.Level(parsedLevel)

	return &Logger{Logger: &l}
}

func (l *Logger) WithRequestID(requestID string) *Logger {
	newLogger := l.With().Str("request_id", requestID).Logger()
	return &Logger{Logger: &newLogger}
}

func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	ctx := l.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	newLogger := ctx.Logger()
	return &Logger{Logger: &newLogger}
}

func (l *Logger) AuditLog(ctx context.Context, action, entityType, entityID string, metadata map[string]interface{}) {
	event := l.Info().
		Str("audit_action", action).
		Str("audit_entity_type", entityType).
		Str("audit_entity_id", entityID)

	if metadata != nil {
		for k, v := range metadata {
			event = event.Interface("audit_"+k, v)
		}
	}

	if requestID := ctx.Value("request_id"); requestID != nil {
		event = event.Str("request_id", requestID.(string))
	}

	event.Msg("Audit log")
}