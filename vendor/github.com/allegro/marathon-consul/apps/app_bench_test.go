package apps

import "testing"

func BenchmarkApp_RegistrationIntents(b *testing.B) {

	// given an average app.
	// Over 80% of our apps has only one port
	// registered and more then five (5) labels but only 2 of them
	// are tags. Over 90% of our application does not have
	// special tags on ports.
	app := &App{
		ID: "app-name",
		Labels: map[string]string{
			"#1":     "label",
			"#2":     "label",
			"consul": "true",
			"#3":     "label",
			"#4":     "tag",
			"#5":     "label",
			"#6":     "tag",
			"#7":     "label",
		},
		PortDefinitions: []PortDefinition{{Labels: map[string]string{"#8": "tag"}}},
	}
	task := &Task{
		Ports: []int{0},
	}

	for i := 0; i < b.N; i++ {
		app.RegistrationIntents(task, "-")
	}
}
