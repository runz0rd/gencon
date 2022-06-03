package gencon

import (
	"os"
	"strings"
	"testing"

	"github.com/c-bata/go-prompt"
	"github.com/c-bata/go-prompt/completer"
)

type desert struct {
	Cake      string
	IceCream  string
	Something struct {
		Else string
	}
}

func (c desert) CakeSuggest(d prompt.Document) []prompt.Suggest {
	return []prompt.Suggest{{Text: "carrot"}, {Text: "chocolate"}}
}

type ExampleConfig struct {
	Desert    desert `yaml:"desert,omitempty" depends:"Dish"`
	Dish      string `yaml:"dish,omitempty"`
	Side      string `yaml:"side" depends:"Dish"`
	Drink     string `yaml:"drink,omitempty" depends:"Dish,Side"`
	Path      string `yaml:"path,omitempty"`
	something struct {
		Else string
	}
}

func (c ExampleConfig) DishSuggest(d prompt.Document) []prompt.Suggest {
	return []prompt.Suggest{{Text: "meat"}, {Text: "fish"}}
}

func (c ExampleConfig) SideSuggest(d prompt.Document) []prompt.Suggest {
	switch c.Dish {
	case "meat":
		return []prompt.Suggest{{Text: "potatoes"}, {Text: "kale"}}
	case "fish":
		return []prompt.Suggest{{Text: "chips"}, {Text: "spinach"}}
	}
	return nil
}

func (c ExampleConfig) PathSuggest(d prompt.Document) []prompt.Suggest {
	completer := completer.FilePathCompleter{
		IgnoreCase: true,
		Filter: func(fi os.FileInfo) bool {
			return fi.IsDir() || strings.HasSuffix(fi.Name(), ".go")
		},
	}
	return completer.Complete(d)
}

func TestWizard_runTags(t *testing.T) {
	type args struct {
		base string
		c    interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"test", args{"", &ExampleConfig{}}, false},
	}
	w := New(
		prompt.OptionShowCompletionAtStart(),
		prompt.OptionCompletionWordSeparator(completer.FilePathCompletionSeparator))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := w.runTags(tt.args.base, tt.args.c); (err != nil) != tt.wantErr {
				t.Errorf("Wizard.runTags() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
