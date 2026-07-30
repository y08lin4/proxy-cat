//go:build linux

package system

func NewDefault() SystemProxy { return NewLinux() }
