package courier

import (
	"fmt"
	"strings"
	"time"

	"github.com/nyaruka/gocommon/dates"
	"github.com/nyaruka/gocommon/httpx"
	"github.com/nyaruka/gocommon/stringsx"
	"github.com/nyaruka/gocommon/uuids"
)

// ChannelLogUUID is our type for a channel log UUID
type ChannelLogUUID uuids.UUID

// ChannelLogType is the type of channel interaction we are logging
type ChannelLogType string

const (
	ChannelLogTypeUnknown         ChannelLogType = "unknown"
	ChannelLogTypeMsgSend         ChannelLogType = "msg_send"
	ChannelLogTypeMsgStatus       ChannelLogType = "msg_status"
	ChannelLogTypeMsgReceive      ChannelLogType = "msg_receive"
	ChannelLogTypeEventReceive    ChannelLogType = "event_receive"
	ChannelLogTypeMultiReceive    ChannelLogType = "multi_receive"
	ChannelLogTypeAttachmentFetch ChannelLogType = "attachment_fetch"
	ChannelLogTypeTokenRefresh    ChannelLogType = "token_refresh"
	ChannelLogTypePageSubscribe   ChannelLogType = "page_subscribe"
	ChannelLogTypeWebhookVerify   ChannelLogType = "webhook_verify"
)

type ChannelError struct {
	code    string
	extCode string
	message string
}

func NewChannelError(code, extCode, message string, args ...any) *ChannelError {
	return &ChannelError{code: code, extCode: extCode, message: fmt.Sprintf(message, args...)}
}

func ErrorResponseStatusCode() *ChannelError {
	return NewChannelError("response_status_code", "", "Unexpected response status code.")
}

func ErrorResponseUnparseable(format string) *ChannelError {
	return NewChannelError("response_unparseable", "", "Unable to parse response as %s.", format)
}

func ErrorResponseUnexpected(expected string) *ChannelError {
	return NewChannelError("response_unexpected", "", "Expected response to be '%s'.", expected)
}

func ErrorResponseValueMissing(key string) *ChannelError {
	return NewChannelError("response_value_missing", "", "Unable to find '%s' response.", key)
}

func ErrorResponseValueUnexpected(key string, expected ...string) *ChannelError {
	es := make([]string, len(expected))
	for i := range expected {
		es[i] = fmt.Sprintf("'%s'", expected[i])
	}
	return NewChannelError("response_value_unexpected", "", "Expected '%s' in response to be %s.", key, strings.Join(es, " or "))
}

func ErrorMediaUnsupported(contentType string) *ChannelError {
	return NewChannelError("media_unsupported", "", "Unsupported attachment media type: %s.", contentType)
}

func ErrorAttachmentNotDecodable() *ChannelError {
	return NewChannelError("attachment_not_decodable", "", "Unable to decode embedded attachment data.")
}

func ErrorExternal(code, message string) *ChannelError {
	if message == "" {
		message = fmt.Sprintf("Service specific error: %s.", code)
	}
	return NewChannelError("external", code, message)
}

func (e *ChannelError) Redact(r stringsx.Redactor) *ChannelError {
	return &ChannelError{code: e.code, extCode: e.extCode, message: r(e.message)}
}

func (e *ChannelError) Message() string {
	return e.message
}

func (e *ChannelError) Code() string {
	return e.code
}

func (e *ChannelError) ExtCode() string {
	return e.extCode
}

// ChannelLog stores the HTTP traces and errors generated by an interaction with a channel.
type ChannelLog struct {
	uuid      ChannelLogUUID
	type_     ChannelLogType
	channel   Channel
	httpLogs  []*httpx.Log
	errors    []*ChannelError
	createdOn time.Time
	elapsed   time.Duration

	attached bool
	recorder *httpx.Recorder
	redactor stringsx.Redactor
}

// NewChannelLogForIncoming creates a new channel log for an incoming request, the type of which won't be known
// until the handler completes.
func NewChannelLogForIncoming(logType ChannelLogType, ch Channel, r *httpx.Recorder, redactVals []string) *ChannelLog {
	return newChannelLog(logType, ch, r, false, redactVals)
}

// NewChannelLogForSend creates a new channel log for a message send
func NewChannelLogForSend(msg Msg, redactVals []string) *ChannelLog {
	return newChannelLog(ChannelLogTypeMsgSend, msg.Channel(), nil, true, redactVals)
}

// NewChannelLogForSend creates a new channel log for an attachment fetch
func NewChannelLogForAttachmentFetch(ch Channel, redactVals []string) *ChannelLog {
	return newChannelLog(ChannelLogTypeAttachmentFetch, ch, nil, true, redactVals)
}

// NewChannelLog creates a new channel log with the given type and channel
func NewChannelLog(t ChannelLogType, ch Channel, redactVals []string) *ChannelLog {
	return newChannelLog(t, ch, nil, false, redactVals)
}

func newChannelLog(t ChannelLogType, ch Channel, r *httpx.Recorder, attached bool, redactVals []string) *ChannelLog {
	return &ChannelLog{
		uuid:      ChannelLogUUID(uuids.New()),
		type_:     t,
		channel:   ch,
		recorder:  r,
		createdOn: dates.Now(),

		redactor: stringsx.NewRedactor("**********", redactVals...),
	}
}

// HTTP logs an outgoing HTTP request and response
func (l *ChannelLog) HTTP(t *httpx.Trace) {
	l.httpLogs = append(l.httpLogs, l.traceToLog(t))
}

func (l *ChannelLog) Error(e *ChannelError) {
	l.errors = append(l.errors, e.Redact(l.redactor))
}

// Deprecated: channel handlers should add user-facing error messages via .Error() instead
func (l *ChannelLog) RawError(err error) {
	l.Error(NewChannelError("", "", err.Error()))
}

func (l *ChannelLog) End() {
	if l.recorder != nil {
		// prepend so it's the first HTTP request in the log
		l.httpLogs = append([]*httpx.Log{l.traceToLog(l.recorder.Trace)}, l.httpLogs...)
	}

	l.elapsed = time.Since(l.createdOn)
}

func (l *ChannelLog) UUID() ChannelLogUUID {
	return l.uuid
}

func (l *ChannelLog) Type() ChannelLogType {
	return l.type_
}

func (l *ChannelLog) SetType(t ChannelLogType) {
	l.type_ = t
}

func (l *ChannelLog) Channel() Channel {
	return l.channel
}

func (l *ChannelLog) Attached() bool {
	return l.attached
}

func (l *ChannelLog) SetAttached(a bool) {
	l.attached = a
}

func (l *ChannelLog) HTTPLogs() []*httpx.Log {
	return l.httpLogs
}

func (l *ChannelLog) Errors() []*ChannelError {
	return l.errors
}

func (l *ChannelLog) CreatedOn() time.Time {
	return l.createdOn
}

func (l *ChannelLog) Elapsed() time.Duration {
	return l.elapsed
}

// if we have an error or a non 2XX/3XX http response then log is considered an error
func (l *ChannelLog) IsError() bool {
	if len(l.errors) > 0 {
		return true
	}

	for _, l := range l.httpLogs {
		if l.StatusCode < 200 || l.StatusCode >= 400 {
			return true
		}
	}

	return false
}

func (l *ChannelLog) traceToLog(t *httpx.Trace) *httpx.Log {
	return httpx.NewLog(t, 2048, 50000, l.redactor)
}
