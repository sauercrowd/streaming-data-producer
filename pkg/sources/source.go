package source

// Source is an abstract Source which could be fed into a sink
type Source interface {
	Wait() // wait the appropriate time to not exceed usage limits, if needed
	AddSink(chan Entry)
}

type Entry struct {
}
