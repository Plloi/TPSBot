package turnips

import (
	"testing"

	"github.com/Plloi/Junior/router"
)

func TestSetup(t *testing.T) {
	type args struct {
		r *router.CommandRouter
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Setup Works?",
			args: args{r: router.NewCommandRouter()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Setup(tt.args.r)
		})
	}
}
