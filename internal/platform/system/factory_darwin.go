//go:build darwin

package system

func NewDefault() SystemProxy { return NewDarwin() }
