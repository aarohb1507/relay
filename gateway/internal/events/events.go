package events

import "net/http"

var Clients = make(map[string]http.ResponseWriter)
