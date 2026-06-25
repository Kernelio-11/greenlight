package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"

	"greenlight.alexedwards.net/internal/validator"
)

type envelope map[string]interface{}

func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(dst)
	if err != nil {

		var syntaxerror *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidunmarshalError *json.InvalidUnmarshalError

		switch {
		case errors.As(err, &syntaxerror):
			return fmt.Errorf(
				"Body conatains badly-formed JSON (at character %d)",
				syntaxerror.Offset,
			)
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("Body contained a badly-formed JSON")
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf(
					"The Body contains incorrect JSON type for Field %q ",
					&unmarshalTypeError.Field,
				)
			}
			return fmt.Errorf(
				"The Body contains incorrect JSON Type (at character %d)",
				unmarshalTypeError.Field,
			)

		case errors.Is(err, io.EOF):
			return errors.New("The Body Must not be empty ")

		case strings.HasPrefix(err.Error(), "json:unknown field"):
			fieldname := strings.TrimPrefix(err.Error(), "json: unknown field")
			return fmt.Errorf("Body contains unknown key %s", fieldname)

		case err.Error() == "http: Request Body too large":
			return fmt.Errorf("Body must be not large than %d bytes", maxBytes)

		case errors.As(err, &invalidunmarshalError):
			panic(err)

		default:
			return err

		}

	}

	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must only contain a single JSON value")
	}

	return nil
}

func (app *application) writeJSON(
	w http.ResponseWriter,
	status int,
	data envelope,
	headers http.Header,
) error {
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}

	js = append(js, '\n')

	for key, value := range headers {
		w.Header()[key] = value
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil
}

func (app *application) ReadIDparam(r *http.Request) (int64, error) {
	params := httprouter.ParamsFromContext(r.Context())
	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)

	if err != nil || id < 1 {
		return 0, errors.New("invalid id parameter")
	}

	return id, nil
}

func (app *application) readString(qs url.Values, key string, default_value string) string {
	s := qs.Get(key)
	if s == "" {
		return default_value
	}
	return s
}

func (app *application) readCSV(qs url.Values, key string, default_value []string) []string {
	csv := qs.Get(key)

	if csv == "" {
		return default_value
	}

	return strings.Split(csv, ",")
}

func (app *application) readInt(
	qs url.Values,
	key string,
	default_value int,
	v *validator.Validator,
) int {
	s := qs.Get(key)

	if s == "" {
		return default_value
	}

	i, err := strconv.Atoi(s)
	if err != nil {
		v.AddError(key, "key must be intger value ")
		return default_value

	}
	return i
}
