# comutex

> Context + Mutex = Comutex

Comutex is a Go package which allows to perform nested calls of functions that lock same mutex 
in the same thread/goroutine with a help of `context.Context`.
Both Mutex and RWMutex are supported.


## License

MIT