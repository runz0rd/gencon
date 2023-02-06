package gencon

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/life4/genesis/slices"
)

const methodSuffix = "Suggest"

type Wizard struct {
	promptOpts []prompt.Option
	skipFilled bool
}

func New(opts ...Option) (*Wizard, error) {
	w := Wizard{}
	for _, opt := range opts {
		if err := opt(&w); err != nil {
			return nil, err
		}
	}
	return &w, nil
}

func (w *Wizard) Prompt(c interface{}, opts ...prompt.Option) error {
	w.promptOpts = opts
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
		sc := make(SuggestCache)
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
			if w.skipFilled && !f.IsZero() {
				continue
			}
			selected, err := w.runSuggest(c, field, fieldPath, sc)
			if err != nil {
				fmt.Println(err)
				continue
			}
			f.SetString(selected)
		case reflect.Int:
			if w.skipFilled {
				continue
			}
			selected, err := w.runSuggest(c, field, fieldPath, sc)
			if err != nil {
				fmt.Println(err)
				continue
			}
			intField, err := strconv.ParseInt(selected, 0, 64)
			if err != nil {
				return err
			}
			f.SetInt(intField)
		case reflect.Bool:
			if w.skipFilled {
				continue
			}
			selected, err := w.runSuggest(c, field, fieldPath, sc)
			if err != nil {
				fmt.Println(err)
				continue
			}
			boolField, err := strconv.ParseBool(selected)
			if err != nil {
				return err
			}
			f.SetBool(boolField)
		}
	}
	return nil
}

func (w Wizard) runSuggest(c interface{}, field, fieldPath string, sc SuggestCache) (string, error) {
	var result string
	fieldCompleter, err := GetFieldCompleter(c, field, fieldPath)
	if err != nil {
		return "", err
	}
	initialValue := GetDefault(c, field)
	result = prompt.Input(fmt.Sprintf("%v> ", fieldPath), func(d prompt.Document) []prompt.Suggest {
		// text := strings.TrimSpace(d.Text)
		lastWord := strings.TrimSpace(d.GetWordBeforeCursor())
		cachedResults := sc.Find(lastWord)
		if len(cachedResults) > 0 && !strings.Contains(lastWord, string(os.PathSeparator)) {
			// if were going through fs, dont offer from cache
			// if already present in cache, skip completer
			return filterSuggestions(lastWord, cachedResults)
		}
		// cache completer results
		sc[lastWord] = fieldCompleter(d)
		return filterSuggestions(lastWord, sc[lastWord])
	}, append(w.promptOpts, prompt.OptionInitialBufferText(initialValue))...)
	if result == "" && !IsOmitempty(c, field) {
		// run input as long as the selection result is "" and the field isnt omitempty
		fmt.Printf("field %q should not be empty (use omitempty tag to avoid this)\n", field)
		return w.runSuggest(c, field, fieldPath, sc)
	}
	return result, nil
}

func GetDependencies(s interface{}, field string) []string {
	return GetTag(s, field, "depends")
}

func GetDefault(s interface{}, field string) string {
	value := ""
	values := GetTag(s, field, "default")
	if len(values) > 0 {
		value = values[0]
	}
	return value
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

func GetFieldCompleter(s interface{}, field, fieldPath string) (prompt.Completer, error) {
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

func IsOmitempty(s interface{}, field string) bool {
	return slices.Contains(GetTag(s, field, "yaml"), "omitempty")
}

type SuggestCache map[string][]prompt.Suggest

// find cached input that is a substring of findKey
func (c SuggestCache) Find(findKey string) []prompt.Suggest {
	var current string
	for cachedKey := range c {
		if strings.Contains(cachedKey, findKey) {
			// if fuzzy.Match(v, key) {
			if len(cachedKey) > len(current) {
				current = cachedKey
			}
		}
	}
	return c[current]
}

func filterSuggestions(text string, suggestions []prompt.Suggest) []prompt.Suggest {
	if strings.Contains(text, string(os.PathSeparator)) {
		// do not filter while completing fs
		return suggestions
	}
	return prompt.FilterContains(suggestions, text, true)
}
