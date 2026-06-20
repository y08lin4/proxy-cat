package core

import (
	"reflect"
	"testing"
)

func TestMihomoArgs(t *testing.T) {
	conf := MihomoLaunchConfig{
		BinaryPath: `C:\Proxy-Cat\mihomo.exe`,
		ConfigPath: `C:\Proxy-Cat\profiles\active\config.yaml`,
		HomeDir:    `C:\Proxy-Cat\mihomo`,
	}

	got := mihomoArgs(conf)
	want := []string{"-f", conf.ConfigPath, "-d", conf.HomeDir}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mihomoArgs() = %#v, want %#v", got, want)
	}
}

func TestMihomoLaunchConfigValidate(t *testing.T) {
	tests := []struct {
		name string
		conf MihomoLaunchConfig
	}{
		{name: "missing binary", conf: MihomoLaunchConfig{ConfigPath: "config.yaml", HomeDir: "mihomo"}},
		{name: "missing config", conf: MihomoLaunchConfig{BinaryPath: "mihomo.exe", HomeDir: "mihomo"}},
		{name: "missing home", conf: MihomoLaunchConfig{BinaryPath: "mihomo.exe", ConfigPath: "config.yaml"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.conf.validate(); err == nil {
				t.Fatal("validate() error = nil, want error")
			}
		})
	}
}
