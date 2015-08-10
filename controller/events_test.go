package main

import (
	"encoding/json"
	"strings"
	"time"

	. "github.com/flynn/flynn/Godeps/_workspace/src/github.com/flynn/go-check"
	cc "github.com/flynn/flynn/controller/client"
	ct "github.com/flynn/flynn/controller/types"
)

func (s *S) TestEvents(c *C) {
	app1 := s.createTestApp(c, &ct.App{Name: "app1"})
	app2 := s.createTestApp(c, &ct.App{Name: "app2"})
	release := s.createTestRelease(c, &ct.Release{})

	jobID1 := "host-job1"
	jobID2 := "host-job2"
	jobID3 := "host-job3"
	jobs := []*ct.Job{
		{ID: jobID1, AppID: app1.ID, ReleaseID: release.ID, Type: "web", State: "starting"},
		{ID: jobID1, AppID: app1.ID, ReleaseID: release.ID, Type: "web", State: "up"},
		{ID: jobID2, AppID: app1.ID, ReleaseID: release.ID, Type: "web", State: "starting"},
		{ID: jobID2, AppID: app1.ID, ReleaseID: release.ID, Type: "web", State: "up"},
		{ID: jobID3, AppID: app2.ID, ReleaseID: release.ID, Type: "web", State: "starting"},
		{ID: jobID3, AppID: app2.ID, ReleaseID: release.ID, Type: "web", State: "up"},
	}

	listener := newEventListener(&EventRepo{db: s.hc.db})
	c.Assert(listener.Listen(), IsNil)

	// sub1 should receive job events for app1, job1
	sub1, err := listener.Subscribe(app1.ID, []string{string(ct.EventTypeJob)}, jobID1)
	c.Assert(err, IsNil)
	defer sub1.Close()

	// sub2 should receive all job events for app1
	sub2, err := listener.Subscribe(app1.ID, []string{string(ct.EventTypeJob)}, "")
	c.Assert(err, IsNil)
	defer sub2.Close()

	// sub3 should receive all job events for app2
	sub3, err := listener.Subscribe(app2.ID, []string{}, "")
	c.Assert(err, IsNil)
	defer sub3.Close()

	for _, job := range jobs {
		s.createTestJob(c, job)
	}

	assertJobEvents := func(sub *EventSubscriber, expected []*ct.Job) {
		var index int
		for {
			select {
			case e, ok := <-sub.Events:
				if !ok {
					c.Fatalf("unexpected close of event stream: %s", sub.Err)
				}
				var jobEvent ct.JobEvent
				c.Assert(json.Unmarshal(e.Data, &jobEvent), IsNil)
				job := expected[index]
				c.Assert(jobEvent, DeepEquals, ct.JobEvent{
					JobID:     job.ID,
					AppID:     job.AppID,
					ReleaseID: job.ReleaseID,
					Type:      job.Type,
					State:     job.State,
				})
				index += 1
				if index == len(expected) {
					return
				}
			case <-time.After(10 * time.Second):
				c.Fatal("timed out waiting for app event")
			}
		}
	}
	assertJobEvents(sub1, jobs[0:2])
	assertJobEvents(sub2, jobs[0:4])
	assertJobEvents(sub3, jobs[4:6])
}

func (s *S) TestStreamAppLifeCycleEvents(c *C) {
	release := s.createTestRelease(c, &ct.Release{})

	events := make(chan *ct.Event)
	stream, err := s.c.StreamEvents(cc.StreamEventsOptions{}, events)
	c.Assert(err, IsNil)
	defer stream.Close()

	app := s.createTestApp(c, &ct.App{Name: "app3"})

	c.Assert(s.c.SetAppRelease(app.ID, release.ID), IsNil)
	newStrategy := "one-by-one"
	c.Assert(s.c.UpdateApp(&ct.App{
		ID:       app.ID,
		Strategy: newStrategy,
	}), IsNil)
	newMeta := map[string]string{
		"foo": "bar",
	}
	c.Assert(s.c.UpdateApp(&ct.App{
		ID:   app.ID,
		Meta: newMeta,
	}), IsNil)

	eventAssertions := []func(*ct.App){
		func(a *ct.App) {
			c.Assert(a.ReleaseID, Equals, app.ReleaseID)
			c.Assert(a.Strategy, Equals, app.Strategy)
			c.Assert(a.Meta, DeepEquals, app.Meta)
		},
		func(a *ct.App) {
			c.Assert(a.ReleaseID, Equals, release.ID)
			c.Assert(a.Strategy, Equals, app.Strategy)
			c.Assert(a.Meta, DeepEquals, app.Meta)
		},
		func(a *ct.App) {
			c.Assert(a.Strategy, Equals, newStrategy)
			c.Assert(a.Meta, DeepEquals, app.Meta)
		},
		func(a *ct.App) {
			c.Assert(a.Strategy, Equals, newStrategy)
			c.Assert(a.Meta, DeepEquals, newMeta)
		},
	}

	for i, fn := range eventAssertions {
		select {
		case e, ok := <-events:
			if !ok {
				c.Fatal("unexpected close of event stream")
			}
			var eventApp *ct.App
			c.Assert(json.Unmarshal(e.Data, &eventApp), IsNil)
			c.Assert(e.AppID, Equals, app.ID)
			c.Assert(e.ObjectType, Equals, ct.EventTypeApp)
			c.Assert(e.ObjectID, Equals, app.ID)
			c.Assert(eventApp, NotNil)
			c.Assert(eventApp.ID, Equals, app.ID)
			fn(eventApp)
		case <-time.After(10 * time.Second):
			c.Fatalf("Timed out waiting for event %d", i)
		}
	}
}

func (s *S) TestStreamReleaseEvents(c *C) {
	events := make(chan *ct.Event)
	stream, err := s.c.StreamEvents(cc.StreamEventsOptions{}, events)
	c.Assert(err, IsNil)
	defer stream.Close()

	release := s.createTestRelease(c, &ct.Release{})

	select {
	case e, ok := <-events:
		if !ok {
			c.Fatal("unexpected close of event stream")
		}
		var eventArtifact *ct.Artifact
		c.Assert(json.Unmarshal(e.Data, &eventArtifact), IsNil)
		c.Assert(e.AppID, Equals, "")
		c.Assert(e.ObjectType, Equals, ct.EventTypeArtifact)
		c.Assert(e.ObjectID, Equals, release.ArtifactID)
		c.Assert(eventArtifact, NotNil)
		c.Assert(eventArtifact.ID, Equals, release.ArtifactID)
	case <-time.After(10 * time.Second):
		c.Fatal("Timed out waiting for artifact event")
	}

	select {
	case e, ok := <-events:
		if !ok {
			c.Fatal("unexpected close of event stream")
		}
		var eventRelease *ct.Release
		c.Assert(json.Unmarshal(e.Data, &eventRelease), IsNil)
		c.Assert(e.AppID, Equals, "")
		c.Assert(e.ObjectType, Equals, ct.EventTypeRelease)
		c.Assert(e.ObjectID, Equals, release.ID)
		c.Assert(eventRelease, DeepEquals, release)
	case <-time.After(10 * time.Second):
		c.Fatal("Timed out waiting for release event")
	}
}

func (s *S) TestStreamFormationEvents(c *C) {
	release := s.createTestRelease(c, &ct.Release{
		Processes: map[string]ct.ProcessType{"foo": {}},
	})
	app := s.createTestApp(c, &ct.App{Name: "stream-formation-test"})

	events := make(chan *ct.Event)
	stream, err := s.c.StreamEvents(cc.StreamEventsOptions{
		ObjectTypes: []ct.EventType{ct.EventTypeScale},
	}, events)
	c.Assert(err, IsNil)
	defer stream.Close()

	formation := s.createTestFormation(c, &ct.Formation{
		AppID:     app.ID,
		ReleaseID: release.ID,
		Processes: map[string]int{"foo": 1},
	})

	select {
	case e, ok := <-events:
		if !ok {
			c.Fatal("unexpected close of event stream")
		}
		var eventProcesses map[string]int
		c.Assert(json.Unmarshal(e.Data, &eventProcesses), IsNil)
		c.Assert(e.AppID, Equals, app.ID)
		c.Assert(e.ObjectType, Equals, ct.EventTypeScale)
		c.Assert(e.ObjectID, Equals, strings.Join([]string{app.ID, release.ID}, ":"))
		c.Assert(eventProcesses, DeepEquals, formation.Processes)
	case <-time.After(10 * time.Second):
		c.Fatal("Timed out waiting for scale event")
	}

	c.Assert(s.c.DeleteFormation(app.ID, release.ID), IsNil)

	select {
	case e, ok := <-events:
		if !ok {
			c.Fatal("unexpected close of event stream")
		}
		var eventProcesses map[string]int
		c.Assert(json.Unmarshal(e.Data, &eventProcesses), IsNil)
		c.Assert(e.AppID, Equals, app.ID)
		c.Assert(e.ObjectType, Equals, ct.EventTypeScale)
		c.Assert(e.ObjectID, Equals, strings.Join([]string{app.ID, release.ID}, ":"))
		c.Assert(eventProcesses, IsNil)
	case <-time.After(10 * time.Second):
		c.Fatal("Timed out waiting for scale event")
	}
}

func (s *S) TestStreamProviderEvents(c *C) {
	events := make(chan *ct.Event)
	stream, err := s.c.StreamEvents(cc.StreamEventsOptions{}, events)
	c.Assert(err, IsNil)
	defer stream.Close()

	provider := s.createTestProvider(c, &ct.Provider{
		URL:  "https://test-stream-provider.example.com",
		Name: "test-stream-provider",
	})

	select {
	case e, ok := <-events:
		if !ok {
			c.Fatal("unexpected close of event stream")
		}
		var eventProvider *ct.Provider
		c.Assert(json.Unmarshal(e.Data, &eventProvider), IsNil)
		c.Assert(e.AppID, Equals, "")
		c.Assert(e.ObjectType, Equals, ct.EventTypeProvider)
		c.Assert(e.ObjectID, Equals, provider.ID)
		c.Assert(eventProvider, DeepEquals, provider)
	case <-time.After(10 * time.Second):
		c.Fatal("Timed out waiting for provider event")
	}
}

func (s *S) TestStreamResourceEvents(c *C) {
	app := s.createTestApp(c, &ct.App{Name: "app4"})

	events := make(chan *ct.Event)
	stream, err := s.c.StreamEvents(cc.StreamEventsOptions{
		ObjectTypes: []ct.EventType{
			ct.EventTypeResource,
			ct.EventTypeResourceDeletion,
		},
	}, events)
	c.Assert(err, IsNil)
	defer stream.Close()

	resource, provider, srv := s.provisionTestResourceWithServer(c, "stream-resources", []string{app.ID})
	defer srv.Close()

	select {
	case e, ok := <-events:
		if !ok {
			c.Fatal("unexpected close of event stream")
		}
		var eventResource *ct.Resource
		c.Assert(json.Unmarshal(e.Data, &eventResource), IsNil)
		c.Assert(e.AppID, Equals, app.ID)
		c.Assert(e.ObjectType, Equals, ct.EventTypeResource)
		c.Assert(e.ObjectID, Equals, strings.Join([]string{provider.ID, resource.ID}, ":"))
		c.Assert(eventResource, DeepEquals, resource)
	case <-time.After(10 * time.Second):
		c.Fatal("Timed out waiting for resource event")
	}

	c.Assert(s.c.DeleteResource(provider.ID, resource.ID), IsNil)

	select {
	case e, ok := <-events:
		if !ok {
			c.Fatal("unexpected close of event stream")
		}
		var eventResource *ct.Resource
		c.Assert(json.Unmarshal(e.Data, &eventResource), IsNil)
		c.Assert(e.AppID, Equals, app.ID)
		c.Assert(e.ObjectType, Equals, ct.EventTypeResourceDeletion)
		c.Assert(e.ObjectID, Equals, strings.Join([]string{provider.ID, resource.ID}, ":"))
		c.Assert(eventResource, DeepEquals, resource)
	case <-time.After(10 * time.Second):
		c.Fatal("Timed out waiting for resource_deletion event")
	}
}

func (s *S) TestStreamKeyEvents(c *C) {
	events := make(chan *ct.Event)
	stream, err := s.c.StreamEvents(cc.StreamEventsOptions{}, events)
	c.Assert(err, IsNil)
	defer stream.Close()

	key := s.createTestKey(c, "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDQfd2OwubRIYVu0lxK0WQVVr/PqcpUBB0xA0P0L6GVsqgL3cKWlFH71r5P3lQuHcuaoKegIP1McNKxp79GixEf/qUdhE0sGS1sIhiPh+t21UMOXPo9oGxDqygAkOZrQiqBU+bDuZaXOMu3epoIMWxvA5WL1wePpKCAnr3PldAGN8I1FPHFu8T3W03PVx4xkZ4CffgZM7q8nXQr1OQC+HM/CC9vmNR8/wkHp0nj1aGgm8Rtc2s2NdGXSB2HJpfeTo1u0fGHCPq9AKrw+ZGmihfes9INENglps8w09FTsPyHXlcVMqO+3c02wBBwuB78EEU9cLDwYK1uRhAV38+eQm/B")

	select {
	case e, ok := <-events:
		if !ok {
			c.Fatal("unexpected close of event stream")
		}
		var eventKey *ct.Key
		c.Assert(json.Unmarshal(e.Data, &eventKey), IsNil)
		c.Assert(e.AppID, Equals, "")
		c.Assert(e.ObjectType, Equals, ct.EventTypeKey)
		c.Assert(e.ObjectID, Equals, key.ID)
		c.Assert(eventKey, DeepEquals, key)
	case <-time.After(10 * time.Second):
		c.Fatal("Timed out waiting for key event")
	}

	c.Assert(s.c.DeleteKey(key.ID), IsNil)

	select {
	case e, ok := <-events:
		if !ok {
			c.Fatal("unexpected close of event stream")
		}
		var eventKey *ct.Key
		c.Assert(json.Unmarshal(e.Data, &eventKey), IsNil)
		c.Assert(e.AppID, Equals, "")
		c.Assert(e.ObjectType, Equals, ct.EventTypeKeyDeletion)
		c.Assert(e.ObjectID, Equals, key.ID)
		c.Assert(eventKey, DeepEquals, key)
	case <-time.After(10 * time.Second):
		c.Fatal("Timed out waiting for key_deletion event")
	}
}
