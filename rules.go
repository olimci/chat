package main

type Rules struct {
	hasPassword bool
	password    string

	hasWelcomeMessage bool
	welcomeMessage    string

	noCommands bool
	noMessages bool
	keepOpen   bool
}

func NewRules() *Rules {
	return &Rules{
		hasPassword: false,
		password:    "",
	}
}

func (r *Rules) Password(password string) *Rules {
	r.hasPassword = true
	r.password = password

	return r
}

func (r *Rules) WelcomeMessage(message string) *Rules {
	r.hasWelcomeMessage = true
	r.welcomeMessage = message

	return r
}

func (r *Rules) NoCommands() *Rules {
	r.noCommands = true

	return r
}

func (r *Rules) NoMessages() *Rules {
	r.noMessages = true

	return r
}

func (r *Rules) KeepOpen() *Rules {
	r.keepOpen = true

	return r
}
