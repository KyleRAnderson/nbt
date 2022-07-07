package iterator

type NopClose struct{}

func (NopClose) Close() {}
