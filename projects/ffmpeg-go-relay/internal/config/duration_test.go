package config

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDurationUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{
			name:  "seconds",
			input: `"30s"`,
			want:  30 * time.Second,
		},
		{
			name:  "minutes",
			input: `"5m"`,
			want:  5 * time.Minute,
		},
		{
			name:  "hours",
			input: `"2h"`,
			want:  2 * time.Hour,
		},
		{
			name:  "milliseconds",
			input: `"500ms"`,
			want:  500 * time.Millisecond,
		},
		{
			name:  "microseconds",
			input: `"100us"`,
			want:  100 * time.Microsecond,
		},
		{
			name:  "nanoseconds",
			input: `"1000ns"`,
			want:  1000 * time.Nanosecond,
		},
		{
			name:    "invalid duration",
			input:   `"invalid"`,
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   `""`,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			input:   `invalid`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Duration
			err := json.Unmarshal([]byte(tt.input), &d)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && time.Duration(d) != tt.want {
				t.Errorf("UnmarshalJSON() got %v, want %v", time.Duration(d), tt.want)
			}
		})
	}
}

func TestDurationMarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		d    Duration
		want string
	}{
		{
			name: "30 seconds",
			d:    Duration(30 * time.Second),
			want: `"30s"`,
		},
		{
			name: "5 minutes",
			d:    Duration(5 * time.Minute),
			want: `"5m0s"`,
		},
		{
			name: "500 milliseconds",
			d:    Duration(500 * time.Millisecond),
			want: `"500ms"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.d)
			if err != nil {
				t.Errorf("MarshalJSON() error = %v", err)
				return
			}
			if string(data) != tt.want {
				t.Errorf("MarshalJSON() got %s, want %s", string(data), tt.want)
			}
		})
	}
}

func TestDurationString(t *testing.T) {
	d := Duration(30 * time.Second)
	want := "30s"
	if d.String() != want {
		t.Errorf("String() got %v, want %v", d.String(), want)
	}
}

func TestDurationAsDuration(t *testing.T) {
	d := Duration(30 * time.Second)
	td := d.AsDuration()
	if td != 30*time.Second {
		t.Errorf("AsDuration() got %v, want %v", td, 30*time.Second)
	}
}

func TestDurationRoundTrip(t *testing.T) {
	durations := []Duration{
		Duration(0),
		Duration(30 * time.Second),
		Duration(5 * time.Minute),
		Duration(2 * time.Hour),
		Duration(500 * time.Millisecond),
	}

	for _, original := range durations {
		// Marshal to JSON
		data, err := json.Marshal(original)
		if err != nil {
			t.Errorf("Marshal failed: %v", err)
			continue
		}

		// Unmarshal back
		var restored Duration
		err = json.Unmarshal(data, &restored)
		if err != nil {
			t.Errorf("Unmarshal failed: %v", err)
			continue
		}

		if original != restored {
			t.Errorf("Round trip failed: %v != %v", original, restored)
		}
	}
}
