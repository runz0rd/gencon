package gencon

type Option func(w *Wizard) error

func OptionSkipFilled() Option {
	return func(w *Wizard) error {
		w.skipFilled = true
		return nil
	}
}
