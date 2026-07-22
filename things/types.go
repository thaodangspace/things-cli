package things

// TypeValue and StatusValue retain the CLI's accepted filter aliases without
// coupling the automation backend to Things' private numeric schema.
func TypeValue(s string) (string, bool) {
	switch s {
	case "to-do", "todo", "task":
		return "to-do", true
	case "project":
		return "project", true
	case "heading":
		return "heading", true
	default:
		return "", false
	}
}

func StatusValue(s string) (string, bool) {
	switch s {
	case "open":
		return "open", true
	case "canceled", "cancelled":
		return "canceled", true
	case "completed", "done":
		return "completed", true
	default:
		return "", false
	}
}
