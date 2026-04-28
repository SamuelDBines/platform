SHELL := /bin/bash
GOTOOLCHAIN ?= local

help:
	@echo "make bishop-co-tech"


build-bishop-co-tech:
	cd backend && GOTOOLCHAIN=$(GOTOOLCHAIN) go build -o bin/bishop-co-tech ./cmd/bishop-co-tech.co.uk

run-bishop-co-tech: build-bishop-co-tech
	./backend/bin/bishop-co-tech

bishop-co-tech: build-bishop-co-tech run-bishop-co-tech
