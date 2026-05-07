package config

const displayNameAnnotation = "yc-scheduler/display-name"

// ToSchedule converts a manifest document into runtime schedule configuration.
func (m ScheduleManifest) ToSchedule() Schedule {
	displayName := m.Metadata.Name
	if value := m.Metadata.Annotations[displayNameAnnotation]; value != "" {
		displayName = value
	}

	return Schedule{
		Name:        m.Metadata.Name,
		DisplayName: displayName,
		Type:        m.Spec.Type,
		Actions:     m.Spec.Actions,
		CronJob:     m.Spec.CronJob,
		DailyJob:    m.Spec.DailyJob,
		WeeklyJob:   m.Spec.WeeklyJob,
		MonthlyJob:  m.Spec.MonthlyJob,
		Resource:    m.Spec.Resource,
	}
}
