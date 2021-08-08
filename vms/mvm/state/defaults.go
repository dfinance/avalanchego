package state

var (
	writeSetsDBPrefix = []byte("writeSets")
	singletonDBPrefix = []byte("singleton")

	initializedKey = []byte("initialized")

	keyDelimiter = []byte(":")
)
