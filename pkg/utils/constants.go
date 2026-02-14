package utils

// will use this for response messages later
type ResponseMessage string

const (
	// users
	UserRegistered ResponseMessage = "User registered successfully"
	UserLoggedIn   ResponseMessage = "User logged in successfully"
	UserUpdated    ResponseMessage = "User updated successfully"
	UserDeleted    ResponseMessage = "User deleted successfully"
	UserRetrieved  ResponseMessage = "User retrieved successfully"

	// monitors
	MonitorCreated   ResponseMessage = "Monitor created successfully"
	MonitorUpdated   ResponseMessage = "Monitor updated successfully"
	MonitorDeleted   ResponseMessage = "Monitor deleted successfully"
	MonitorRetrieved ResponseMessage = "Monitor retrieved successfully"
)

func (rm ResponseMessage) String() string {
	return string(rm)
}

func BuildNotFoundMessage(resource string) ResponseMessage {
	return ResponseMessage(resource + " not found")
}
