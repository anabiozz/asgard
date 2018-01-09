package filter

type Filter interface {
	Match(string) bool
}
