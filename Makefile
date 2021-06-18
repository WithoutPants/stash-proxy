# TODO - to suppress Windows console, uncomment this
#LDFLAGS := -H=windowsgui
EXTRA_LDFLAGS := -extldflags=-static -s -w

build:
	go build -v -tags "osusergo netgo" -ldflags "$(LDFLAGS) $(EXTRA_LDFLAGS)"
