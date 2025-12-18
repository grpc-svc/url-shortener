package storage

import "errors"

var (
	errURLNotFound = errors.New("URL not found")
	errURLExists   = errors.New("URL already exists")
)
