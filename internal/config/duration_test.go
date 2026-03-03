package config

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// TestDurationUnmarshalYAML_String tests unmarshalling Duration from string format.
func TestDurationUnmarshalYAML_String(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    time.Duration
		wantErr bool
	}{
		{
			name:    "30 minutes",
			yaml:    "timeout: \"30m\"",
			want:    30 * time.Minute,
			wantErr: false,
		},
		{
			name:    "1 hour 30 minutes",
			yaml:    "timeout: \"1h30m\"",
			want:    90 * time.Minute,
			wantErr: false,
		},
		{
			name:    "300 seconds",
			yaml:    "timeout: \"300s\"",
			want:    300 * time.Second,
			wantErr: false,
		},
		{
			name:    "5 minutes",
			yaml:    "timeout: \"5m\"",
			want:    5 * time.Minute,
			wantErr: false,
		},
		{
			name:    "invalid format",
			yaml:    "timeout: \"invalid\"",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type testStruct struct {
				Timeout Duration `yaml:"timeout"`
			}

			var ts testStruct
			err := yaml.Unmarshal([]byte(tt.yaml), &ts)

			if (err != nil) != tt.wantErr {
				t.Errorf("unmarshal error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && ts.Timeout.Duration != tt.want {
				t.Errorf("Timeout = %v, want %v", ts.Timeout.Duration, tt.want)
			}
		})
	}
}

// TestDurationUnmarshalYAML_Int tests unmarshalling Duration from integer format (seconds).
func TestDurationUnmarshalYAML_Int(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    time.Duration
		wantErr bool
	}{
		{
			name:    "300 seconds as integer",
			yaml:    "timeout: 300",
			want:    300 * time.Second,
			wantErr: false,
		},
		{
			name:    "zero duration",
			yaml:    "timeout: 0",
			want:    0,
			wantErr: false,
		},
		{
			name:    "1800 seconds (30 minutes)",
			yaml:    "timeout: 1800",
			want:    1800 * time.Second,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type testStruct struct {
				Timeout Duration `yaml:"timeout"`
			}

			var ts testStruct
			err := yaml.Unmarshal([]byte(tt.yaml), &ts)

			if (err != nil) != tt.wantErr {
				t.Errorf("unmarshal error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && ts.Timeout.Duration != tt.want {
				t.Errorf("Timeout = %v, want %v", ts.Timeout.Duration, tt.want)
			}
		})
	}
}

// TestDurationIsZero tests the IsZero method.
func TestDurationIsZero(t *testing.T) {
	tests := []struct {
		name     string
		duration Duration
		want     bool
	}{
		{
			name:     "zero duration",
			duration: Duration{0},
			want:     true,
		},
		{
			name:     "non-zero duration (30 minutes)",
			duration: Duration{30 * time.Minute},
			want:     false,
		},
		{
			name:     "non-zero duration (1 second)",
			duration: Duration{1 * time.Second},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.duration.IsZero()
			if got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDurationMarshalYAML tests marshalling Duration to YAML.
func TestDurationMarshalYAML(t *testing.T) {
	tests := []struct {
		name     string
		duration Duration
		want     string
	}{
		{
			name:     "30 minutes",
			duration: Duration{30 * time.Minute},
			want:     "30m0s",
		},
		{
			name:     "1 hour 30 minutes",
			duration: Duration{90 * time.Minute},
			want:     "1h30m0s",
		},
		{
			name:     "5 minutes",
			duration: Duration{5 * time.Minute},
			want:     "5m0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type testStruct struct {
				Timeout Duration `yaml:"timeout"`
			}

			ts := testStruct{Timeout: tt.duration}
			data, err := yaml.Marshal(ts)
			if err != nil {
				t.Errorf("marshal error = %v", err)
				return
			}

			// Unmarshal to verify round-trip
			var ts2 testStruct
			err = yaml.Unmarshal(data, &ts2)
			if err != nil {
				t.Errorf("unmarshal error = %v", err)
				return
			}

			if ts2.Timeout.Duration != tt.duration.Duration {
				t.Errorf("round-trip: Timeout = %v, want %v", ts2.Timeout.Duration, tt.duration.Duration)
			}
		})
	}
}

// TestDurationString tests the String method.
func TestDurationString(t *testing.T) {
	tests := []struct {
		name     string
		duration Duration
		want     string
	}{
		{
			name:     "30 minutes",
			duration: Duration{30 * time.Minute},
			want:     "30m0s",
		},
		{
			name:     "1 hour",
			duration: Duration{1 * time.Hour},
			want:     "1h0m0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.duration.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}
