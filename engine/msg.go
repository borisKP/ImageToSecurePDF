package main

import (
	"errors"
	"fmt"
	"strings"
)

// Msg is a language-neutral event. The UI owns every translation and can
// re-render the whole log history after a language switch.
type Msg struct {
	Key  string   `json:"key"`
	Args []string `json:"args,omitempty"`
	Text string   `json:"-"`
}

func M(key, fallback string, args ...string) Msg {
	return Msg{Key: key, Args: append([]string{}, args...), Text: fallback}
}

func (m Msg) localize() string {
	t := m.Text
	for i, a := range m.Args {
		t = strings.ReplaceAll(t, fmt.Sprintf("{%d}", i), a)
	}
	return t
}

func A(v ...any) []string {
	out := make([]string, 0, len(v))
	for _, x := range v {
		out = append(out, fmt.Sprint(x))
	}
	return out
}

type MsgError struct {
	msg   Msg
	cause error
}

func (e *MsgError) Error() string { return e.msg.localize() }
func (e *MsgError) Unwrap() error { return e.cause }

func E(key, fallback string, args ...string) error {
	return &MsgError{msg: M(key, fallback, args...)}
}

func EW(key, fallback string, cause error, args ...string) error {
	return &MsgError{msg: M(key, fallback, args...), cause: cause}
}

func asMsg(err error) (Msg, bool) {
	var me *MsgError
	if errors.As(err, &me) {
		return me.msg, true
	}
	return Msg{}, false
}
