package model

import "time"

func (sd ServerDefinition) Clone() ServerDefinition {
	schedule := make([]*Event, 0, len(sd.Schedule))
	for _, event := range sd.Schedule {
		schedule = append(schedule, &Event{
			ID:   event.ID,
			Day:  event.Day,
			Hour: event.Hour,
			Kind: event.Kind,
		})
	}
	tags := make([]*Tag, 0, len(sd.Tags))
	for _, tag := range sd.Tags {
		tags = append(tags, &Tag{
			ID:          tag.ID,
			Description: tag.Description,
			Icon:        tag.Icon,
			Value:       tag.Value,
		})
	}
	moderators := make([]*Moderator, 0, len(sd.Moderators))
	for _, moderator := range sd.Moderators {
		moderators = append(moderators, &Moderator{
			ID:      moderator.ID,
			SteamID: moderator.SteamID,
		})
	}
	cloned := sd
	cloned.Tags = tags
	cloned.Schedule = schedule
	cloned.Moderators = moderators
	return cloned
}

func (d *ServerDefinition) Scrub() {
	d.ID = "id"
	d.InstanceID = "instanceId"
	d.AllocationID = "allocationId"
	d.ElasticIP = "elasticIp"

	for i := range d.Schedule {
		d.Schedule[i].ID = "id"
	}
	for i := range d.Tags {
		d.Tags[i].ID = "id"
	}
	for i := range d.Moderators {
		d.Moderators[i].ID = "id"
	}
}

func (s LiveServer) Clone() LiveServer {
	return s
}

func (s *LiveServer) Scrub() {
	s.ID = "id"
	s.Definition.Scrub()
	s.AssociationID = "associationId"
	s.UpdatedAt = time.Time{}
	s.CreatedAt = time.Time{}
}

func (s DormantServer) Clone() DormantServer {
	return s
}

func (s *DormantServer) Scrub() {
	s.ID = "id"
	s.Definition.Scrub()
	s.UpdatedAt = time.Time{}
	s.CreatedAt = time.Time{}
}

func (s ArchivedServer) Clone() ArchivedServer {
	return s
}

func (s *ArchivedServer) Scrub() {
	s.ID = "id"
	s.Definition.Scrub()
	s.UpdatedAt = time.Time{}
	s.CreatedAt = time.Time{}
}
