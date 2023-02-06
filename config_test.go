package gencon

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/c-bata/go-prompt"
	"github.com/c-bata/go-prompt/completer"
	"github.com/davecgh/go-spew/spew"
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
	Desert        desert `yaml:"desert,omitempty" depends:"Dish"`
	Dish          string `yaml:"dish,omitempty"`
	Side          string `yaml:"side" depends:"Dish" default:"tt"`
	Drink         string `yaml:"drink,omitempty" depends:"Dish,Side"`
	Path          string `yaml:"path,omitempty" default:"asd"`
	AmIRight      bool   `yaml:"am_i_right,omitempty"`
	AlreadyFilled string `yaml:"already_filled,omitempty"`
	RealSlow      string `yaml:"real_slow,omitempty"`
	something     struct {
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

func (c ExampleConfig) AlreadyFilledSuggest(d prompt.Document) []prompt.Suggest {
	return []prompt.Suggest{{Text: "this should be skipped"}}
}

func (c ExampleConfig) AmIRightSuggest(d prompt.Document) []prompt.Suggest {
	return []prompt.Suggest{{Text: "true"}, {Text: "0"}}
}

func (c ExampleConfig) RealSlowSuggest(d prompt.Document) []prompt.Suggest {
	time.Sleep(1 * time.Second)
	return []prompt.Suggest{{Text: "true"}, {Text: "0"}}
}

func TestWizard_runTags(t *testing.T) {
	type args struct {
		base string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"test", args{""}, false},
	}
	w, err := New(OptionSkipFilled())
	if err != nil {
		t.Fatal(err)
	}
	w.promptOpts = []prompt.Option{
		prompt.OptionShowCompletionAtStart(),
		prompt.OptionCompletionWordSeparator(completer.FilePathCompletionSeparator),
	}
	for _, tt := range tests {
		c := &ExampleConfig{}
		c.AlreadyFilled = "heh"
		t.Run(tt.name, func(t *testing.T) {
			if err := w.runTags(tt.args.base, c); (err != nil) != tt.wantErr {
				t.Errorf("Wizard.runTags() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
		spew.Dump(c)
	}
}
