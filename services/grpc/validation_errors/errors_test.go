package validation_errors

import (
	"errors"
	"testing"

	ov "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type (
	testRequest struct {
		Name string
		Tags []string
	}
)

func (r *testRequest) Validate() error {
	return ov.ValidateStruct(r,
		ov.Field(&r.Name, ov.Required, ov.Length(0, 10), is.Alphanumeric),
		ov.Field(&r.Tags, ov.Each(
			ov.Length(1, 10),
			is.Alphanumeric,
		), ov.Length(0, 1)),
	)
}

func TestFromValidation(t *testing.T) {
	data := &testRequest{
		Name: "dsklsdjfln#$^^@",
		Tags: []string{"asdsa^&", "sdfsd&%sdf", "ewyh324$^#$#"},
	}

	test := []*errdetails.BadRequest_FieldViolation{
		{Field: "Name", Description: "the length must be no more than 10"},
		{Field: "Tags.0", Description: "must contain English letters and digits only"},
		{Field: "Tags.1", Description: "must contain English letters and digits only"},
		{Field: "Tags.2", Description: "the length must be between 1 and 10"},
	}

	err := FromValidation(data.Validate())
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	for _, detail := range status.Convert(err).Details() {
		require.IsType(t, new(errdetails.BadRequest), detail)
		switch val := detail.(type) {
		case *errdetails.BadRequest:
			require.ElementsMatch(t, test, val.GetFieldViolations())
		}
	}
}

func TestFromValidationOtherError(t *testing.T) {
	testError := errors.New("test error")
	err := FromValidation(testError)
	require.Error(t, err)
	require.Equal(t, testError, err)

	err = FromValidation(nil)
	require.Nil(t, err)
}
