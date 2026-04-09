package main

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/distr-sh/distr/internal/util"
	composeapi "github.com/docker/compose/v5/pkg/api"
)

type composeEventProcessor struct {
	updateStatus    func(string)
	mut             sync.Mutex
	imageBlobs      map[string]map[string]bool // image -> blob ID -> done
	pulledImages    map[string]bool            // images that finished pulling
	containerStatus map[string]string          // container ID -> status text
}

func NewEventProcessor(updateStatus func(string)) *composeEventProcessor {
	return &composeEventProcessor{
		updateStatus:    updateStatus,
		imageBlobs:      make(map[string]map[string]bool),
		pulledImages:    make(map[string]bool),
		containerStatus: make(map[string]string),
	}
}

func (p *composeEventProcessor) Start(_ context.Context, _ string) {}

func (p *composeEventProcessor) Done(_ string, _ bool) {}

func (p *composeEventProcessor) On(events ...composeapi.Resource) {
	p.mut.Lock()
	defer p.mut.Unlock()
	for _, e := range events {
		switch e.Text {
		case composeapi.StatusPulling:
			image := strings.TrimPrefix(e.ID, "Image ")
			if _, ok := p.imageBlobs[image]; !ok {
				p.imageBlobs[image] = make(map[string]bool)
			}

		case composeapi.StatusPulled:
			image := strings.TrimPrefix(e.ID, "Image ")
			delete(p.imageBlobs, image)
			p.pulledImages[image] = true

		case composeapi.StatusDownloading:
			image := strings.TrimPrefix(e.ParentID, "Image ")
			if _, ok := p.imageBlobs[image]; !ok {
				p.imageBlobs[image] = make(map[string]bool)
			}
			if _, exists := p.imageBlobs[image][e.ID]; !exists {
				p.imageBlobs[image][e.ID] = false
			}

		case composeapi.StatusDownloadComplete:
			image := strings.TrimPrefix(e.ParentID, "Image ")
			if _, ok := p.imageBlobs[image]; !ok {
				p.imageBlobs[image] = make(map[string]bool)
			}
			p.imageBlobs[image][e.ID] = true

		case composeapi.StatusCreating, composeapi.StatusCreated,
			composeapi.StatusStarting, composeapi.StatusStarted,
			composeapi.StatusRunning:
			p.containerStatus[e.ID] = e.Text

		case composeapi.StatusError:
			p.containerStatus[e.ID] = fmt.Sprintf("error: %s", e.Text)

		default:
			continue
		}
	}

	if msg := p.buildMessage(); msg != "" {
		p.updateStatus(msg)
	}
}

func (p *composeEventProcessor) buildMessage() string {
	sb := new(strings.Builder)

	if len(p.containerStatus) > 0 {
		for _, container := range slices.Sorted(maps.Keys(p.containerStatus)) {
			fmt.Fprintf(sb, "%s %s\n", p.containerStatus[container], container)
		}

		return sb.String()
	}

	for _, image := range slices.Sorted(maps.Keys(p.imageBlobs)) {
		blobs := p.imageBlobs[image]
		total := len(blobs)
		if total == 0 {
			fmt.Fprintf(sb, "Pulling %s\n", image)
		} else {
			done := util.SeqLen(util.SeqFilter(maps.Values(blobs), util.Identity))
			fmt.Fprintf(sb, "Pulling %s: %d/%d\n", image, done, total)
		}
	}

	for _, image := range slices.Sorted(maps.Keys(p.pulledImages)) {
		fmt.Fprintf(sb, "Pulling %s: done\n", image)
	}

	return sb.String()
}
