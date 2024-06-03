package controllers

import (
	corev1 "k8s.io/api/core/v1"
	"reflect"
	"testing"
)

func Test_convertBase10ToBase2(t *testing.T) {
	type args struct {
		size float64
		unit string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test 1: Convert 1 KB to KiB",
			args: args{size: 1, unit: "KB"},
			want: "0.98Ki",
		},
		{
			name: "Test 2: Convert 1 MB to MiB",
			args: args{size: 1, unit: "MB"},
			want: "0.95Mi",
		},
		{
			name: "Test 3: Convert 1 GB to GiB",
			args: args{size: 1, unit: "GB"},
			want: "0.93Gi",
		},
		{
			name: "Test 4: Convert 6750 MB to MiB",
			args: args{size: 6750, unit: "MB"},
			want: "6437.30Mi",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConvertBase10ToBase2(tt.args.size, tt.args.unit); got != tt.want {
				t.Errorf("convertBase10ToBase2() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_convertEnvVarSliceToMap(t *testing.T) {
	type args struct {
		envars []corev1.EnvVar
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "Test 1: Convert slice of EnvVar to map",
			args: args{
				envars: []corev1.EnvVar{
					{Name: "Var1", Value: "Value1"},
					{Name: "Var2", Value: "Value2"},
				},
			},
			want: map[string]string{
				"VAR1": "Value1",
				"VAR2": "Value2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertEnvVarSliceToMap(tt.args.envars); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertEnvVarSliceToMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getResourceSize(t *testing.T) {
	type args struct {
		size string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test 1: Convert KB to KiB",
			args: args{
				size: "1KB",
			},
			want: "0.98Ki",
		},
		{
			name: "Test 2: Convert MB to MiB",
			args: args{
				size: "1MB",
			},
			want: "0.95Mi",
		},
		{
			name: "Test 3: Convert GB to GiB",
			args: args{
				size: "1GB",
			},
			want: "0.93Gi",
		},
		{
			name: "Test 4: Convert TB to TiB",
			args: args{
				size: "1TB",
			},
			want: "0.91Ti",
		},
		{
			name: "Test 5: Convert PB to PiB",
			args: args{
				size: "1PB",
			},
			want: "0.89Pi",
		},
		{
			name: "Test 6: Convert Mi to Mi (no conversion)",
			args: args{
				size: "0.95Mi",
			},
			want: "0.95Mi",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getResourceSize(tt.args.size); got != tt.want {
				t.Errorf("getResourceSize() = %v, want %v", got, tt.want)
			}
		})
	}
}
