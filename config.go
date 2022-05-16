package gencon

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/davecgh/go-spew/spew"
)

type SuggestConfig interface {
	Completer(name string, d prompt.Document) []prompt.Suggest
	DepsFilled(name string) bool
}

type Wizard struct {
	opts []prompt.Option
}

func New(opts ...prompt.Option) *Wizard {
	return &Wizard{opts: opts}
}

func (w Wizard) Run(c SuggestConfig) error {
	return w.run("", c)
}

func (w Wizard) run(base string, c SuggestConfig) error {
	rType := reflect.TypeOf(c).Elem()
	var fields []string
	for i := 0; i < rType.NumField(); i++ {
		fields = append(fields, rType.Field(i).Name)
	}
	for i := 0; i < len(fields); i++ {
		field := fields[i]
		fieldPath := strings.TrimPrefix(strings.Join([]string{base, field}, "."), ".")
		// spew.Dump(field)
		if !c.DepsFilled(field) {
			// move to the end
			fields = append(fields[:i], fields[i+1:]...)
			fields = append(fields, field)
			i--
			continue
		}
		f := reflect.ValueOf(c).Elem().FieldByName(field)
		if f.Kind() == reflect.Struct {
			structReflectValue := reflect.New(f.Type())
			structField, ok := structReflectValue.Interface().(SuggestConfig)
			if !ok {
				fmt.Printf("%q does not implement SuggestConfig, skipping\n", fieldPath)
				continue
			}
			if err := w.run(fieldPath, structField); err != nil {
				return err
			}
			f.Set(reflect.ValueOf(structField).Elem())
			continue
		}
		stringField := prompt.Input(fmt.Sprintf("%v> ", fieldPath),
			func(d prompt.Document) []prompt.Suggest {
				return c.Completer(field, d)
			},
			w.opts...)
		switch f.Kind() {
		case reflect.String:
			f.SetString(stringField)
		case reflect.Int:
			intField, err := strconv.ParseInt(stringField, 0, 64)
			if err != nil {
				return err
			}
			f.SetInt(intField)
		}
	}
	return nil
}

func (w Wizard) RunTags(c interface{}) error {
	return w.runTags("", c)
}

func (w Wizard) runTags(base string, c interface{}) error {
	//TODO test for struct and anon struct
	rType := reflect.TypeOf(c).Elem()
	var fields []string
	for i := 0; i < rType.NumField(); i++ {
		fields = append(fields, rType.Field(i).Name)
	}

	for i := 0; i < len(fields); i++ {
		field := fields[i]
		fieldPath := strings.TrimPrefix(strings.Join([]string{base, field}, "."), ".")
		// spew.Dump(field)
		fs, err := GetFieldSuggest(c, field)
		if err != nil {
			fmt.Println(err)
			continue
		}
		if !AreDependenciesFilled(c, GetDependencies(c, field)) {
			// move to the end
			fields = append(fields[:i], fields[i+1:]...)
			fields = append(fields, field)
			i--
			continue
		}
		stringField := prompt.Input(fmt.Sprintf("%v> ", fieldPath), fs, w.opts...)
		f := reflect.ValueOf(c).Elem().FieldByName(field)
		switch f.Kind() {
		case reflect.String:
			f.SetString(stringField)
		case reflect.Int:
			intField, err := strconv.ParseInt(stringField, 0, 64)
			if err != nil {
				return err
			}
			f.SetInt(intField)
		}
	}
	spew.Dump(c)
	return nil
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

func GetFieldSuggest(s interface{}, field string) (prompt.Completer, error) {
	name := fmt.Sprintf("%vSuggest", field)
	method := GetMethod(s, name)
	if !method.IsValid() {
		return nil, fmt.Errorf("method %v is not valid or doesnt exist", name)
	}
	callable, ok := method.Interface().(func(prompt.Document) []prompt.Suggest)
	if !ok {
		return nil, fmt.Errorf("method %v is not of prompt.Completer type", name)
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
