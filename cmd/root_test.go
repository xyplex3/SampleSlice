package cmd

import (
	"testing"
)

func TestVersionCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "version subcommand",
			args:    []string{"version"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("rootCmd.Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRootCommandRequiredFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "no flags",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "input flag with non-existent file",
			args:    []string{"-i", "/nonexistent/file.wav"},
			wantErr: true,
		},
		{
			name:    "input long flag with non-existent file",
			args:    []string{"--input", "/nonexistent/file.wav"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("rootCmd.Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHelpFlag(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "short help flag",
			args:    []string{"-h"},
			wantErr: false,
		},
		{
			name:    "long help flag",
			args:    []string{"--help"},
			wantErr: false,
		},
		{
			name:    "help subcommand",
			args:    []string{"help"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("rootCmd.Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}