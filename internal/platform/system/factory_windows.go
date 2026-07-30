//go:build windows

package system

func NewDefault() SystemProxy { return NewWindows() }
