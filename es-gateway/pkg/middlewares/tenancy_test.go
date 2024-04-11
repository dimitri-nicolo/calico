package middlewares

import (
	"net/http"
	"reflect"
	"testing"
)

func TestKibanaTenancy_Enforce(t *testing.T) {
	type fields struct {
		tenantID string
	}
	tests := []struct {
		name   string
		fields fields
		want   func(next http.Handler) http.Handler
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := KibanaTenancy{
				tenantID: tt.fields.tenantID,
			}
			if got := k.Enforce(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Enforce() = %v, want %v", got, tt.want)
			}
		})
	}
}
