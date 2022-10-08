package config

type Credentials struct {
	EmailAddress string
	Password     string
}

var TapoCredentials = Credentials{
	EmailAddress: "redactedForGitCommit",
	Password:     "redactedForGitCommit",
}
