package gencon

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/c-bata/go-prompt"
)

const methodSuffix = "Suggest"

type Wizard struct {
	opts []prompt.Option
}

func New(opts ...prompt.Option) *Wizard {
	return &Wizard{opts: opts}
}

func (w Wizard) Run(c interface{}) error {
	return w.runTags("", c)
}

func (w Wizard) runTags(base string, c interface{}) error {
	rType := reflect.TypeOf(c).Elem()
	var fields []string
	for _, f := range reflect.VisibleFields(rType) {
		if f.IsExported() {
			fields = append(fields, f.Name)
		}
	}

	fieldOccured := make(map[string]int)
	for i := 0; i < len(fields); i++ {
		field := fields[i]
		fieldOccured[field]++
		if fieldOccured[field] > 2 {
			// check if stuck in an endless loop with unfilled deps
			return fmt.Errorf("field %q has dependencies that cannot be filled", field)
		}

		fieldPath := strings.TrimPrefix(strings.Join([]string{base, field}, "."), ".")
		if !AreDependenciesFilled(c, GetDependencies(c, field)) {
			// move to the end
			fields = append(fields[:i], fields[i+1:]...)
			fields = append(fields, field)
			i--
			continue
		}
		f := reflect.ValueOf(c).Elem().FieldByName(field)
		switch f.Kind() {
		case reflect.Struct:
			// TODO anon struct?
			structReflectValue := reflect.New(f.Type())
			if err := w.runTags(fieldPath, structReflectValue.Interface()); err != nil {
				fmt.Println(err)
				continue
			}
			f.Set(reflect.ValueOf(structReflectValue.Interface()).Elem())
			continue
		case reflect.String:
			stringField, err := w.runSuggest(c, field, fieldPath)
			if err != nil {
				fmt.Println(err)
				continue
			}
			f.SetString(stringField)
		case reflect.Int:
			stringField, err := w.runSuggest(c, field, fieldPath)
			if err != nil {
				fmt.Println(err)
				continue
			}
			intField, err := strconv.ParseInt(stringField, 0, 64)
			if err != nil {
				return err
			}
			f.SetInt(intField)
		}
	}
	// spew.Dump(c)
	return nil
}

func (w Wizard) runSuggest(c interface{}, field, fieldPath string) (string, error) {
	fs, err := GetFieldSuggest(c, field, fieldPath)
	if err != nil {
		return "", err
	}
	return prompt.Input(fmt.Sprintf("%v> ", fieldPath), fs, w.opts...), nil
}

func GetDependencies(s interface{}, field string) []string {
	return GetTag(s, field, "depends")
}

func AreDependenciesFilled(s interface{}, deps []string) bool {
	if len(deps) == 0 {
		return true
	}
	rValue := reflect.ValueOf(s).Elem()
	for _, dep := range deps {
		value := rValue.FieldByName(dep).Interface()
		if value != "" && value != 0 && value != nil {
			continue
		}
		return false
	}
	return true
}

func GetFieldSuggest(s interface{}, field, fieldPath string) (prompt.Completer, error) {
	method := GetMethod(s, field+methodSuffix)
	if !method.IsValid() {
		return nil, fmt.Errorf("method %v is not valid or doesnt exist", fieldPath+methodSuffix)
	}
	callable, ok := method.Interface().(func(prompt.Document) []prompt.Suggest)
	if !ok {
		return nil, fmt.Errorf("method %v does not implement prompt.Completer", fieldPath+methodSuffix)
	}
	return callable, nil
}

func GetTag(s interface{}, field, name string) []string {
	t := reflect.TypeOf(s).Elem()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Name == field {
			tags := f.Tag.Get(name)
			if tags == "" {
				return nil
			}
			return strings.Split(f.Tag.Get(name), ",")
		}
	}
	return nil
}

func GetMethod(s interface{}, method string) reflect.Value {
	return reflect.ValueOf(s).MethodByName(method)
}

func pType(t reflect.Type) reflect.Type {
	for {
		if t.Kind() != reflect.Ptr {
			return t
		}
		t = t.Elem()
	}
}
