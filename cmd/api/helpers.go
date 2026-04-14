package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

type envelope map[string]interface{}

func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	err := json.NewDecoder(r.Body).Decode(dst)
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

		case errors.As(err, &invalidunmarshalError):
			panic(err)

		default:
			return err

		}

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
