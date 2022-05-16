package main

import (
	"fmt"

	"github.com/c-bata/go-prompt"
	"github.com/runz0rd/gencon"
)

type desert struct {
	Cake      string
	IceCream  string
	Something struct {
		Else string
	}
}

func (c desert) Completer(name string, d prompt.Document) []prompt.Suggest {
	switch name {
	case "Cake":
		return []prompt.Suggest{{Text: "lemon"}, {Text: "chocolate"}}
	case "IceCream":
		return []prompt.Suggest{{Text: "vanilla"}, {Text: "chocolate"}}
	}
	return nil
}

func (c desert) DepsFilled(name string) bool {
	switch name {
	case "Cake":
		if c.IceCream != "" {
			return true
		}
		return false
	}
	return true
}

type ExampleConfig struct {
	Desert    desert `yaml:"desert,omitempty" depends:"Dish"`
	Dish      string `yaml:"dish,omitempty"`
	Side      string `yaml:"side,omitempty" depends:"Dish"`
	Drink     string `yaml:"drink,omitempty" depends:"Dish,Side"`
	Something struct {
		Else string
	}
}

func (c ExampleConfig) DishSuggest(d prompt.Document) []prompt.Suggest {
	return []prompt.Suggest{{Text: "meat"}, {Text: "fish"}}
}

func (c ExampleConfig) Completer(name string, d prompt.Document) []prompt.Suggest {
	switch name {
	case "Dish":
		return []prompt.Suggest{{Text: "meat"}, {Text: "fish"}}
	case "Side":
		switch c.Dish {
		case "meat":
			return []prompt.Suggest{{Text: "potatoes"}, {Text: "kale"}}
		case "fish":
			return []prompt.Suggest{{Text: "chips"}, {Text: "spinach"}}
		}
	case "Drink":
		switch c.Dish {
		case "meat":
			return []prompt.Suggest{{Text: "cola"}, {Text: "tea"}}
		case "fish":
			return []prompt.Suggest{{Text: "lemonade"}, {Text: "apple juice"}}
		}
	}
	return nil
}

func (c ExampleConfig) DepsFilled(name string) bool {
	switch name {
	case "Side", "Drink", "Desert":
		if c.Dish != "" {
			return true
		}
		return false
	}
	return true
}

func main() {
	c := ExampleConfig{}
	gencon.New(
		prompt.OptionShowCompletionAtStart(),
		prompt.OptionPrefixTextColor(prompt.Red),
	).RunTags(&c)
	fmt.Println(c)
}
