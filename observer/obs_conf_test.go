package observer

import (
	"testing"

	"github.com/letsencrypt/boulder/cmd"
	blog "github.com/letsencrypt/boulder/log"
)

func TestObsConf_ValidateDebugAddr(t *testing.T) {
	type fields struct {
		DebugAddr string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// valid
		{"valid", fields{":8080"}, false},
		// invalid
		{"out of range high", fields{":65536"}, true},
		{"out of range low", fields{":0"}, true},
		{"not even a port", fields{":foo"}, true},
		{"missing :", fields{"foo"}, true},
		{"missing port", fields{"foo:"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ObsConf{
				DebugAddr: tt.fields.DebugAddr,
			}
			if err := c.ValidateDebugAddr(); (err != nil) != tt.wantErr {
				t.Errorf("ObsConf.ValidateDebugAddr() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestObsConf_validate(t *testing.T) {
	type fields struct {
		Syslog    cmd.SyslogConfig
		DebugAddr string
		MonConfs  []*MonConf
	}
	type args struct {
		log blog.Logger
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ObsConf{
				Syslog:    tt.fields.Syslog,
				DebugAddr: tt.fields.DebugAddr,
				MonConfs:  tt.fields.MonConfs,
			}
			if err := c.validate(tt.args.log); (err != nil) != tt.wantErr {
				t.Errorf("ObsConf.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
