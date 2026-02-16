package config

// ToSchedule converts a manifest document into runtime schedule configuration.
func (m ScheduleManifest) ToSchedule() Schedule {
	return Schedule{
		Name:       m.Metadata.Name,
		Type:       m.Spec.Type,
		Actions:    m.Spec.Actions,
		CronJob:    m.Spec.CronJob,
		DailyJob:   m.Spec.DailyJob,
		WeeklyJob:  m.Spec.WeeklyJob,
		MonthlyJob: m.Spec.MonthlyJob,
		Resource:   m.Spec.Resource,
	}
}
