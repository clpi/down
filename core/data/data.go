package data

type (
	Settings struct {
		Config map[string]interface{ Init() }
	}
)
