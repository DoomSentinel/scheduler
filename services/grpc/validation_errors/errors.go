package validation_errors

import (
	"fmt"

	ov "github.com/go-ozzo/ozzo-validation/v4"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func FromValidation(err error) error {
	if err == nil {
		return nil
	}

	var validationErr ov.Errors
	switch v := err.(type) {
	case ov.Errors:
		validationErr = v
	case ov.InternalError:
		return status.Error(codes.Internal, v.Error())
	default:
		return err
	}

	statusResponse, err := makeResponse(validationErr)
	if err != nil {
		return err
	}

	if statusResponse != nil {
		return statusResponse.Err()
	}

	return nil
}

func makeResponse(errors ov.Errors) (*status.Status, error) {
	response := status.New(codes.InvalidArgument, codes.InvalidArgument.String())

	details := makeBadRequest(errors)
	if len(details.FieldViolations) == 0 {
		return nil, nil
	}

	return response.WithDetails(details)
}

func makeBadRequest(errors ov.Errors) *errdetails.BadRequest {
	return &errdetails.BadRequest{
		FieldViolations: makeFieldViolations(errors, "", make([]*errdetails.BadRequest_FieldViolation, 0)),
	}
}

func makeFieldViolations(errors ov.Errors, parentField string, violations []*errdetails.BadRequest_FieldViolation) []*errdetails.BadRequest_FieldViolation {
	for field, err := range errors {
		switch val := err.(type) {
		case ov.Errors:
			violations = makeFieldViolations(val, joinKeys(parentField, field), violations)
		case ov.ErrorObject, error:
			violations = append(violations, &errdetails.BadRequest_FieldViolation{
				Field:       joinKeys(parentField, field),
				Description: val.Error(),
			})
		}
	}

	return violations
}

func joinKeys(parent, child string) string {
	if parent == "" {
		return child
	}

	return parent + "." + child
}

func ExtractFieldErrors(err error) map[string][]error {
	errMap := make(map[string][]error)

	for _, detail := range status.Convert(err).Details() {
		switch t := detail.(type) {
		case *errdetails.BadRequest:
			for _, violation := range t.GetFieldViolations() {
				fieldErrors, ok := errMap[violation.GetField()]
				if ok {
					errMap[violation.GetField()] = append(fieldErrors, fmt.Errorf(violation.GetDescription()))
				} else {
					errMap[violation.GetField()] = []error{fmt.Errorf(violation.GetDescription())}
				}
			}
		}
	}

	if len(errMap) > 0 {
		return errMap
	}

	return nil
}
