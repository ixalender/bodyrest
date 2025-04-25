package bodyrest

import (
	"encoding/json"
	"log"
	"mime/multipart"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
)

var once sync.Once

const defaultResponse = ""
const logPrefix = "[bodyrest]"

type RestErrorFunc func(w http.ResponseWriter, r *http.Request, status int)

var restErrorFunc RestErrorFunc

func SetRestErrorHandler(errFunc RestErrorFunc) {
	once.Do(func() {
		restErrorFunc = errFunc
	})
}

func HandleTo(handlerFunc interface{}) http.HandlerFunc {

	handlerType := reflect.TypeOf(handlerFunc)
	if handlerType.Kind() != reflect.Func {
		log.Fatal("Handler is not a function")
	}

	if handlerType == reflect.TypeOf(func(http.ResponseWriter, *http.Request) {}) {
		log.Fatal("http.HandlerFunc is not a valid parameter, use interface function instead")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerType := reflect.TypeOf(handlerFunc)
		if handlerType.Kind() != reflect.Func {
			log.Println("handler is not a function")
			http.Error(w, defaultResponse, http.StatusInternalServerError)
			return
		}

		if handlerType.NumIn() <= 0 {
			handlerValue := reflect.ValueOf(handlerFunc)

			results := handlerValue.Call([]reflect.Value{})
			if len(results) != 1 {
				log.Println("handler does not return exactly one value")
				if restErrorFunc != nil {
					restErrorFunc(w, r, http.StatusInternalServerError)
					return
				}

				http.Error(w, defaultResponse, http.StatusInternalServerError)
				return
			}

			handler, ok := results[0].Interface().(http.HandlerFunc)
			if !ok {
				log.Println("handler does not return http.HandlerFunc")
				if restErrorFunc != nil {
					restErrorFunc(w, r, http.StatusInternalServerError)
					return
				}

				http.Error(w, defaultResponse, http.StatusInternalServerError)
				return
			}

			handler.ServeHTTP(w, r)
			return
		}

		if (r.Method == http.MethodPost ||
			r.Method == http.MethodPut ||
			r.Method == http.MethodPatch) &&
			(r.Body == nil || r.ContentLength == 0) {
			log.Printf("request body is empty\n")
			if restErrorFunc != nil {
				restErrorFunc(w, r, http.StatusBadRequest)
				return
			}

			http.Error(w, defaultResponse, http.StatusBadRequest)
			return
		}

		// TODO: extract to check path and handler params on handler definition
		var handlerArgsToCall []reflect.Value = make([]reflect.Value, handlerType.NumIn())
		lastInspectedPathPartIndex := -1
		hasBodyStructParsed := false
		for i := 0; i < handlerType.NumIn(); i++ {
			paramType := handlerType.In(i)
			paramValue := reflect.New(paramType)

			if paramType.Kind() == reflect.Struct {
				if hasBodyStructParsed {
					log.Println("got more than one body struct")
					if restErrorFunc != nil {
						restErrorFunc(w, r, http.StatusBadRequest)
						return
					}

					http.Error(w, defaultResponse, http.StatusBadRequest)
					return
				}

				if paramType == reflect.TypeOf(multipart.Form{}) {
					err := r.ParseMultipartForm(32 << 20)
					if err != nil {
						log.Printf("failed to parse multipart form: %v\n", err)
						if restErrorFunc != nil {
							restErrorFunc(w, r, http.StatusBadRequest)
							return
						}
					}

					paramValue.Elem().Set(reflect.ValueOf(*r.MultipartForm))

				} else {
					err := json.NewDecoder(r.Body).Decode(paramValue.Interface())
					if err != nil {
						log.Printf("failed to parse request body: %v\n", err)
						if restErrorFunc != nil {
							restErrorFunc(w, r, http.StatusBadRequest)
							return
						}

						http.Error(w, defaultResponse, http.StatusBadRequest)
						return
					}

					valid := areRequiredFieldsValid(paramValue.Interface())
					if !valid {
						log.Println("required fields are not valid")
						if restErrorFunc != nil {
							restErrorFunc(w, r, http.StatusBadRequest)
							return
						}

						http.Error(w, defaultResponse, http.StatusBadRequest)
						return
					}
				}

				hasBodyStructParsed = true
				handlerArgsToCall[i] = paramValue.Elem()
			} else {
				routePattern := chi.RouteContext(r.Context()).RoutePattern()

				pathParts := strings.Split(r.URL.Path, "/")
				patternParts := strings.Split(routePattern, "/")

				for idx, part := range patternParts {
					if strings.Contains(part, "{") && strings.Contains(part, "}") && idx > lastInspectedPathPartIndex {
						var pVal interface{}
						var convErr error

						switch paramType.Kind() {
						case reflect.Int:
							pVal, convErr = strconv.Atoi(pathParts[idx])
						case reflect.String:
							pVal = pathParts[idx]
						case reflect.Bool:
							pVal, convErr = strconv.ParseBool(pathParts[idx])
						case reflect.Float64:
							pVal, convErr = strconv.ParseFloat(pathParts[idx], 64)
						}
						if convErr != nil {
							log.Printf("failed to parse path param under index %d: %v\n", idx, convErr)
							if restErrorFunc != nil {
								restErrorFunc(w, r, http.StatusBadRequest)
								return
							}
							http.Error(w, defaultResponse, http.StatusBadRequest)
							return
						}

						paramValue.Elem().Set(reflect.ValueOf(pVal))
						handlerArgsToCall[i] = paramValue.Elem()

						lastInspectedPathPartIndex = idx
						break
					}
				}
			}
		}

		handlerValue := reflect.ValueOf(handlerFunc)

		if handlerType.NumIn() != len(handlerArgsToCall) {
			log.Printf("got %d arguments, expected %d\n", len(handlerArgsToCall), handlerType.NumIn())
			if restErrorFunc != nil {
				restErrorFunc(w, r, http.StatusBadRequest)
				return
			}

			http.Error(w, defaultResponse, http.StatusBadRequest)
			return
		}

		zeroValueArguments := false
		for i := 0; i < handlerType.NumIn(); i++ {
			if !handlerArgsToCall[i].IsValid() {
				zeroValueArguments = true
				break
			}
		}

		if zeroValueArguments {
			log.Println("handler has zero value arguments")
			if restErrorFunc != nil {
				restErrorFunc(w, r, http.StatusBadRequest)
				return
			}

			http.Error(w, defaultResponse, http.StatusBadRequest)
			return
		}
		results := handlerValue.Call(handlerArgsToCall)

		if len(results) != 1 {
			log.Println("handler does not return exactly one value")
			if restErrorFunc != nil {
				restErrorFunc(w, r, http.StatusInternalServerError)
				return
			}

			http.Error(w, defaultResponse, http.StatusInternalServerError)
			return
		}

		handler, ok := results[0].Interface().(http.HandlerFunc)
		if !ok {
			log.Println("handler does not return http.HandlerFunc")
			if restErrorFunc != nil {
				restErrorFunc(w, r, http.StatusInternalServerError)
				return
			}

			http.Error(w, defaultResponse, http.StatusInternalServerError)
			return
		}

		handler.ServeHTTP(w, r)
	})
}

func areRequiredFieldsValid(obj interface{}) bool {
	value := reflect.ValueOf(obj)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	if value.Kind() != reflect.Struct {
		return false
	}

	for i := 0; i < value.NumField(); i++ {
		field := value.Type().Field(i)
		fieldValue := value.Field(i)

		tag := field.Tag.Get("json")

		if tag != "" && tag != "-" && !strings.Contains(tag, "omitempty") {
			if isFieldEmpty(fieldValue) {
				return false
			}
		}
	}

	return true
}

func isFieldEmpty(field reflect.Value) bool {
	switch field.Kind() {
	case reflect.String:
		return field.String() == ""
	case reflect.Array, reflect.Slice, reflect.Map:
		return field.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return field.IsNil()
	default:
		return false
	}
}
